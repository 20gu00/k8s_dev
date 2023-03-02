# Development Guide

# Releases and Official Builds

Official releases are built in Docker containers.  Details are [here](../../build/README.md).  You can do simple builds and development with just a local Docker installation.  If want to build go locally outside of docker, please continue below.

## Go development environment

Kubernetes is written in [Go](http://golang.org) programming language. If you haven't set up Go development environment, please follow [this instruction](http://golang.org/doc/code.html) to install go tool and set up GOPATH. Ensure your version of Go is at least 1.3.

## Put kubernetes into GOPATH

We highly recommend to put kubernetes' code into your GOPATH. For example, the following commands will download kubernetes' code under the current user's GOPATH (Assuming there's only one directory in GOPATH.):

```
$ echo $GOPATH
/home/user/goproj
$ mkdir -p $GOPATH/src/github.com/GoogleCloudPlatform/
$ cd $GOPATH/src/github.com/GoogleCloudPlatform/
$ git clone https://github.com/GoogleCloudPlatform/kubernetes.git
```

The commands above will not work if there are more than one directory in ``$GOPATH``.

(Obviously, clone your own fork of Kubernetes if you plan to do development.)

## godep and dependency management

Kubernetes uses [godep](https://github.com/tools/godep) to manage dependencies. It is not strictly required for building Kubernetes but it is required when managing dependencies under the Godeps/ tree, and is required by a number of the build and test scripts. Please make sure that ``godep`` is installed and in your ``$PATH``.

### Installing godep
There are many ways to build and host go binaries. Here is an easy way to get utilities like ```godep``` installed:

1. Ensure that [mercurial](http://mercurial.selenic.com/wiki/Download) is installed on your system. (some of godep's dependencies use the mercurial
source control system).  Use ```apt-get install mercurial``` or ```yum install mercurial``` on Linux, or [brew.sh](http://brew.sh) on OS X, or download
directly from mercurial.
2. Create a new GOPATH for your tools and install godep:
```
export GOPATH=$HOME/go-tools
mkdir -p $GOPATH
go get github.com/tools/godep
```

3. Add $GOPATH/bin to your path. Typically you'd add this to your ~/.profile:
```
export GOPATH=$HOME/go-tools
export PATH=$PATH:$GOPATH/bin
```

### Using godep
Here is a quick summary of `godep`.  `godep` helps manage third party dependencies by copying known versions into Godeps/_workspace. Here is the recommended way to set up your system. There are other ways that may work, but this is the easiest one I know of.

1. Devote a directory to this endeavor:

```
export KPATH=$HOME/code/kubernetes
mkdir -p $KPATH/src/github.com/GoogleCloudPlatform/kubernetes
cd $KPATH/src/github.com/GoogleCloudPlatform/kubernetes
git clone https://path/to/your/fork .
# Or copy your existing local repo here. IMPORTANT: making a symlink doesn't work.
```

2. Set up your GOPATH.

```
# Option A: this will let your builds see packages that exist elsewhere on your system.
export GOPATH=$KPATH:$GOPATH
# Option B: This will *not* let your local builds see packages that exist elsewhere on your system.
export GOPATH=$KPATH
# Option B is recommended if you're going to mess with the dependencies.
```

3. Populate your new $GOPATH.

```
cd $KPATH/src/github.com/GoogleCloudPlatform/kubernetes
godep restore
```

4. To add a dependency, you can do ```go get path/to/dependency``` as usual.

5. To package up a dependency, do

```
cd $KPATH/src/github.com/GoogleCloudPlatform/kubernetes
godep save ./...
# Sanity check that your Godeps.json file is ok by re-restoring:
godep restore
```

I (lavalamp) have sometimes found it expedient to manually fix the /Godeps/godeps.json file to minimize the changes.

Please send dependency updates in separate commits within your PR, for easier reviewing.

## Hooks

Before committing any changes, please link/copy these hooks into your .git
directory. This will keep you from accidentally committing non-gofmt'd go code.

```
cd kubernetes/.git/hooks/
ln -s ../../hooks/prepare-commit-msg .
ln -s ../../hooks/commit-msg .
```

## Unit tests

```
cd kubernetes
hack/test-go.sh
```

Alternatively, you could also run:

```
cd kubernetes
godep go test ./...
```

If you only want to run unit tests in one package, you could run ``godep go test`` under the package directory. For example, the following commands will run all unit tests in package kubelet:

```
$ cd kubernetes # step into kubernetes' directory.
$ cd pkg/kubelet
$ godep go test
# some output from unit tests
PASS
ok      github.com/GoogleCloudPlatform/kubernetes/pkg/kubelet   0.317s
```

## Coverage
```
cd kubernetes
godep go tool cover -html=target/c.out
```

## Integration tests

You need an [etcd](https://github.com/coreos/etcd/releases/tag/v2.0.0) in your path, please make sure it is installed and in your ``$PATH``.
```
cd kubernetes
hack/test-integration.sh
```

## End-to-End tests

You can run an end-to-end test which will bring up a master and two minions, perform some tests, and then tear everything down. Make sure you have followed the getting started steps for your chosen cloud platform (which might involve changing the `KUBERNETES_PROVIDER` environment variable to something other than "gce".
```
cd kubernetes
hack/e2e-test.sh
```

Pressing control-C should result in an orderly shutdown but if something goes wrong and you still have some VMs running you can force a cleanup with this command:
```
go run hack/e2e.go --down
```

### Flag options
See the flag definitions in `hack/e2e.go` for more options, such as reusing an existing cluster, here is an overview:

```sh
# Build binaries for testing
go run hack/e2e.go --build

# Create a fresh cluster.  Deletes a cluster first, if it exists
go run hack/e2e.go --up

# Create a fresh cluster at a specific release version.
go run hack/e2e.go --up --version=0.7.0

# Test if a cluster is up.
go run hack/e2e.go --isup

# Push code to an existing cluster
go run hack/e2e.go --push

# Push to an existing cluster, or bring up a cluster if it's down.
go run hack/e2e.go --pushup

# Run all tests
go run hack/e2e.go --test

# Run tests matching a glob.
go run hack/e2e.go --tests=...
```

### Combining flags
```sh
# Flags can be combined, and their actions will take place in this order:
# -build, -push|-up|-pushup, -test|-tests=..., -down
# e.g.:
go run e2e.go -build -pushup -test -down

# -v (verbose) can be added if you want streaming output instead of only
# seeing the output of failed commands.

# -ctl can be used to quickly call kubectl against your e2e cluster. Useful for
# cleaning up after a failed test or viewing logs.
go run e2e.go -ctl='get events'
go run e2e.go -ctl='delete pod foobar'
```

## Testing out flaky tests
[Instructions here](flaky-tests.md)

## Add/Update dependencies

Kubernetes uses [godep](https://github.com/tools/godep) to manage dependencies. To add or update a package, please follow the instructions on [godep's document](https://github.com/tools/godep).

To add a new package ``foo/bar``:

- Make sure the kubernetes' root directory is in $GOPATH/github.com/GoogleCloudPlatform/kubernetes
- Run ``godep restore`` to make sure you have all dependancies pulled.
- Download foo/bar into the first directory in GOPATH: ``go get foo/bar``.
- Change code in kubernetes to use ``foo/bar``.
- Run ``godep save ./...`` under kubernetes' root directory.

To update a package ``foo/bar``:

- Make sure the kubernetes' root directory is in $GOPATH/github.com/GoogleCloudPlatform/kubernetes
- Run ``godep restore`` to make sure you have all dependancies pulled.
- Update the package with ``go get -u foo/bar``.
- Change code in kubernetes accordingly if necessary.
- Run ``godep update foo/bar`` under kubernetes' root directory.

## Keeping your development fork in sync

One time after cloning your forked repo:

```
git remote add upstream https://github.com/GoogleCloudPlatform/kubernetes.git
```

Then each time you want to sync to upstream:

```
git fetch upstream
git rebase upstream/master
```

## Regenerating the CLI documentation

```
hack/run-gendocs.sh
```
