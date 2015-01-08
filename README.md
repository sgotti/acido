# Acido - App Container Image/Rocket Utility and Testbed


Acido is an utility and testbed to manipulate App Container Images.
I don't know if it will become a real project or just a testbed to implement features for Rocket.

I started with the idea to build Images with a simple build language (like a DockerFile, but only the build parts as the remaining parts are implemented by the appc spec).

For doing this some basic features were needed:
* Extract aci images satisfying all dependencies
* Create new aci images containing only the changes from a base image
* Copy and extract files inside the container (like Dockerfile ADD and COPY commands).
* Executes commands inside the container (like Dockerfile RUN command)

By now the first 2 point are implemented and you can create and manage new images starting from a base image.

After the image discovery and fetching mechanism (appc/spec#16 point 1) is completed the need to use the hash in the image dependencies will be removed (the right way is to use image discovery, the actual implementation is not spec compliant).

I hope that some parts of this project will be useful to the rocket project (tar package new features and fixes, image renderer, fsdiffer)


## Examples

### Create a base Fedora 21 image

```
$ wget 'https://github.com/fedora-cloud/docker-brew-fedora/blob/b252b53a976a0e908805d59fb7250e8d5072f4e8/fedora-21-release.tar.xz?raw=true' -O /tmp/fedora-21-release.tar.xz

mkdir -p /tmp/fedora21/rootfs
cd /tmp/fedora21/rootfs
tar xvf /tmp/fedora-21-release.tar.xz

```

Create a manifest file for this aci:
```
$ cat /tmp/fedora21/manifest
{
    "acKind": "ImageManifest",
    "acVersion": "0.1.0",
    "name": "example.com/fedora",
    "labels": [
        {
            "name": "version",
            "value": "21.0.0"
        },
        {
            "name": "arch",
            "value": "amd64"
        },
        {
            "name": "os",
            "value": "linux"
        }
    ]
}
```

### Build the image.

```
$ ./acido build /tmp/fedora21/ /tmp/fedora21.aci
```

### Import the image to the Rocket Store
```
$ ./acido import /tmp/fedora21.aci
INFO import.go:47: image: /tmp/fedora21.aci, hash: sha256-41bcf35ec05a1f08d2240bcc300dcf4b016e4ae332d399a5e958827edb1640fb
```

The returned hash value will be used in the next operations (until the discovery mechanism is implemented)

### Start a new build using the previous image as a base
```
$ ./acido startbuild sha256-41bcf35ec05a1f08d2240bcc300dcf4b016e4ae332d399a5e958827edb1640fb
INFO startbuild.go:39: tmpdir: /tmp/645016202
INFO startbuild.go:52: Image extracted to /tmp/645016202
```

This will extract the requested image and create a base manifest with the dependencies already defined to the extracted image.


### Do your changes
```
$systemd-nspawn -D /tmp/645016202/rootfs /bin/bash 
Spawning container rootfs on /tmp/167589206/rootfs.
Press ^] three times within 1s to kill container.


$ yum -y install grep
[...]

$ yum -y remove firewalld
[...]

Container rootfs terminated by signal KILL.
```

### Complete the build
```
./acido build /tmp/645016202 /tmp/fedora21-new.aci 
```

A new image will be created containing only the diffs from the dependencies.


### Test the new image

```
./acido import /tmp/fedora21-new.aci 
INFO import.go:47: image: /tmp/fedora21-new.aci, hash: sha256-bffedaa3154dc6e871380a9e880745c08e88db1f614c61292ba1f512c76444a7
```

```
./acido extract sha256-bffedaa3154dc6e871380a9e880745c08e88db1f614c61292ba1f512c76444a7
```

You can test with a diff -r -q that the new extracted image directory rebuilt from its dependencies matches the directory used by the build in the previous step.
