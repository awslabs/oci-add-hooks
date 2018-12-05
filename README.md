## Oci Add Hooks

oci-add-hooks is an OCI runtime with the sole purpose of injecting OCI
`prestart`, `poststart`, and `poststop` hooks into a container `config.json` before
passing along to an OCI compatable runtime.

## Usage

This runtime can be invoked by doing

```
oci-add-hooks \
	--hook-config-path </path/to/hook/config>
	--runtime-path </path/to/oci/runtime> \
	…\
	[--bundle <path/to/bundle> \]
	…
```
- `hook-config-path` is a json file that follows the format described [here](https://github.com/opencontainers/runtime-spec/blob/master/config.md#posix-platform-hooks).
- `runtime-path` is a path to an OCI runtime binary.
- `bundle`,if present, specifies the path to the bundle directory.

### With Docker

A few things need to be done to use `oci-add-hooks` with Docker. First modify
`/etc/docker/daemon.json` to includ a "runtimes" section similiar to the following:

```json
{
  "runtimes": {
    "oci-add-hook": {
      "path": "oci-add-hooks",
      "runtimeArgs": ["--hook-config-path",
	"/path/to/config.json",
	"--runtime-path",
	"<path/to/oci/runtime>"]
    }
  }
}
```
> note: path here should either include this binaries name when it's on the path
> or the full path/name if it's not.


If we had a hypothetical hook config located at `/home/user/hook-config.json`

```json
{
  "hooks": {
    "prestart": [
      {
        "path": "path/to/prestart/hook",
        "args": ["hook", "some", "args", "here"]
      }
    ]
  }
}
```

and we wanted to launch containers with runc our /etc/docker/daemon.json would
look like:

```json
{
  "runtimes": {
    "oci-add-hooks": {
      "path": "oci-add-hooks",
      "runtimeArgs": ["--hook-config-path",
	"/home/user/hook-config.json",
	"--runtime-path",
	"runc"]
    }
  }
}
```

This is assuming that both `oci-add-hooks` and `runc` are in the path. You
can restart Docker to trigger a reload of this config file. You should be able
to verify it has this runtime by doing `docker info` and seeing something like:

```
…
Runtimes: oci-add-hooks runc
…
```

Once this is setup you can use this runtime (and the configured hooks) by
doing a `docker run` command and adding the --runtime=oci-add-hooks flag.
`docker run --rm --runtime=oci-add-hooks <image>`

## What is it doing

When invoked as above, `oci-add-hooks` will parse the file specified by
`hook-config-path` as specified in the [runtime-spec](https://github.com/opencontainers/runtime-spec/blob/master/config.md#posix-platform-hooks)
section on POSIX-platform hooks. It will merge these hooks into the config.json
file located at the path passed to `bundle`, writing the changes back to disk.
If hooks are already present in the spec, it will pre-pend these hooks to the
existing ones. It will then strip out the options and args that are specific to
oci-add-hooks and passthrough to the binary pointed at by `runtime-path`.

## License

This library is licensed under the Apache 2.0 License. 
