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

By now with the `extract` and `startbuild` commands the image is fetched from the Rocket CAS store by ImageID. In future there will be a way to get an image also by its name and labels (coreos/rocket#395).

Note: By now rocket is waiting for PR to implements various features:
 * An acirenderer that is able to build an image with dependencies (coreos/rocket#464, coreos/rocket#465). So, as today, generated ACIs that have dependencies cannot be used.
 * A way to get images from the cas by an ImageID or by a app name and optional labels (coreos/rocket#392, coreos/rocket#393, coreos/rocket#394, coreos/rocket#395, coreos/rocket#322). 

As of now, some parts of this program have been proposed for coreos/rocket and appc/spec (tar package new features and fixes, acirenderer etc...), other are going to be proposed (ACIBuilder, FSDiffer etc...)


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
INFO import.go:47: image: /tmp/fedora21.aci, hash: sha512-34e79f60f57fe90951612975651562349ac5be20bef2ba8f9dd4900794d1647c
```

The returned hash value will be used in the next operations (until the discovery mechanism is implemented)

### Start a new build using the previous image as a base
```
$ ./acido startbuild sha512-34e79f60f57fe90951612975651562349ac5be20bef2ba8f9dd4900794d1647c
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
INFO import.go:47: image: /tmp/fedora21-new.aci, hash: sha512-7faaf487ee92c4b8efb4a2148a98866de601cdac8f4ddb519e6213e4c3d52c4e
```

```
./acido extract sha512-7faaf487ee92c4b8efb4a2148a98866de601cdac8f4ddb519e6213e4c3d52c4e
```

You can test with a diff -r -q that the new extracted image directory rebuilt from its dependencies matches the directory used by the build in the previous step.
