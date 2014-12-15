package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/coreos/fleet/log"
	"github.com/docker/docker/pkg/system"
)

type insecureLinkError error

// ExtractTar extracts a tarball (from a tar.Reader) into the given directory
func ExtractTar(tr *tar.Reader, dir string, overwrite bool, pathWhitelist map[string]uint8) error {
	um := syscall.Umask(0)
	defer syscall.Umask(um)

	dirhdrs := []*tar.Header{}
Tar:
	for {
		hdr, err := tr.Next()
		switch err {
		case io.EOF:
			break Tar
		case nil:
			if len(pathWhitelist) > 0 {
				p := filepath.Clean(hdr.Name)
				// Check only files inside rootfs
				if strings.HasPrefix(p, "rootfs/") {
					relpath, err := filepath.Rel("rootfs/", hdr.Name)
					if err != nil {
						return err
					}
					//log.Infof("relpath: %s", relpath)

					if _, ok := pathWhitelist[relpath]; !ok {
						log.Infof("file: %s not in pathWhitelist", hdr.Name)
						continue
					}
				}
			}
			err = ExtractEntry(tr, hdr, dir, overwrite)
			if err != nil {
				return fmt.Errorf("error extracting tarball: %v", err)
			}
			if hdr.Typeflag == tar.TypeDir {
				dirhdrs = append(dirhdrs, hdr)
			}

		default:
			return fmt.Errorf("error extracting tarball: %v", err)
		}
	}

	// Restore dirs atime and mtime. This has to be done after extracting
	// as a file extraction will change its parent directory's times.
	for _, hdr := range dirhdrs {
		p := filepath.Join(dir, hdr.Name)
		ts := []syscall.Timespec{timeToTimespec(hdr.AccessTime), timeToTimespec(hdr.ModTime)}
		if err := syscall.UtimesNano(p, ts); err != nil {
			return err
		}
	}
	return nil
}

func ExtractFile(tr *tar.Reader, file string) ([]byte, error) {
	for {
		hdr, err := tr.Next()
		switch err {
		case io.EOF:
			return nil, fmt.Errorf("File not found")
		case nil:
			if filepath.Clean(hdr.Name) != filepath.Clean(file) {
				continue
			}
			if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
				return nil, fmt.Errorf("Requested file not a regular file")
			}
			buf, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("error extracting tarball: %v", err)
			}
			return buf, nil
		default:
			return nil, fmt.Errorf("error extracting tarball: %v", err)
		}
	}
}

func ExtractEntry(tr *tar.Reader, hdr *tar.Header, dir string, overwrite bool) error {
	p := filepath.Join(dir, hdr.Name)
	fi := hdr.FileInfo()
	typ := hdr.Typeflag

	if overwrite {
		// If file exists remove it
		info, err := os.Lstat(p)
		if err == nil {
			//and if it's a dir remove it and all his childs
			if info.IsDir() && typ != tar.TypeDir {
				log.V(1).Infof("%s: Parent was a dir and the new file isn't a dir\n", p)
				err := os.RemoveAll(p)
				if err != nil {
					return err
				}
			}
			if !info.IsDir() && typ != tar.TypeDir {
				log.V(1).Infof("%s: Parent wasn't a dir and the new file isn't a dir\n", p)
				err := os.Remove(p)
				if err != nil {
					return err
				}
			}
		}
	}
	// Create parent dir if it doesn't exists
	pdir := filepath.Dir(p)
	_, err := os.Lstat(pdir)
	if err != nil {
		if err := os.MkdirAll(pdir, 0755); err != nil {
			return err
		}
	}

	switch {
	case typ == tar.TypeReg || typ == tar.TypeRegA:

		f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, fi.Mode())
		if err != nil {
			f.Close()
			return err
		}
		_, err = io.Copy(f, tr)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()
	case typ == tar.TypeDir:
		// If it already exists just change permissions
		_, err := os.Lstat(p)
		if err != nil {
			if err := os.MkdirAll(p, fi.Mode()); err != nil {
				return err
			}
		} else {
			if err := os.Chmod(p, fi.Mode()); err != nil {
				return err
			}
		}
	case typ == tar.TypeLink:
		dest := filepath.Join(dir, hdr.Linkname)
		if !strings.HasPrefix(dest, dir) {
			return insecureLinkError(fmt.Errorf("insecure link %q -> %q", p, hdr.Linkname))
		}
		if err := os.Link(dest, p); err != nil {
			return err
		}
	case typ == tar.TypeSymlink:
		dest := filepath.Join(filepath.Dir(p), hdr.Linkname)
		if !strings.HasPrefix(dest, dir) {
			return insecureLinkError(fmt.Errorf("insecure symlink %q -> %q", p, hdr.Linkname))
		}
		if err := os.Symlink(hdr.Linkname, p); err != nil {
			return err
		}
	case typ == tar.TypeChar:
		dev := makedev(int(hdr.Devmajor), int(hdr.Devminor))
		mode := uint32(fi.Mode()) | syscall.S_IFCHR
		if err := syscall.Mknod(p, mode, dev); err != nil {
			return err
		}
	case typ == tar.TypeBlock:
		dev := makedev(int(hdr.Devmajor), int(hdr.Devminor))
		mode := uint32(fi.Mode()) | syscall.S_IFBLK
		if err := syscall.Mknod(p, mode, dev); err != nil {
			return err
		}

	// TODO(jonboulle): implement other modes
	default:
		return fmt.Errorf("unsupported type: %v", typ)
	}

	// Restore file atime and mtime.
	ts := []syscall.Timespec{timeToTimespec(hdr.AccessTime), timeToTimespec(hdr.ModTime)}
	if hdr.Typeflag != tar.TypeSymlink {
		if err := system.UtimesNano(p, ts); err != nil && err != system.ErrNotSupportedPlatform {
			return err
		}
	} else {
		if err := system.LUtimesNano(p, ts); err != nil && err != system.ErrNotSupportedPlatform {
			return err
		}
	}
	return nil
}

// makedev mimics glib's gnu_dev_makedev
func makedev(major, minor int) int {
	return (minor & 0xff) | (major & 0xfff << 8) | int((uint64(minor & ^0xff) << 12)) | int(uint64(major & ^0xfff)<<32)
}
