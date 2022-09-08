# Pip Install Cloud Native Buildpack
The Paketo Buildpack for Pip Install is a Cloud Native Buildpack that installs
packages using pip and makes it available to the application.

The buildpack is published for consumption at
`gcr.io/paketo-buildpacks/pip-install` and `paketobuildpacks/pip-install`.

## Behavior
This buildpack participates if `requirements.txt` exists at the root the app.

The buildpack will do the following:
* At build time:
  - Installs the application packages to a layer made available to the app.
  - Prepends the layer site-packages onto `PYTHONPATH`.
  - If a vendor directory is available, will attempt to run `pip install` in an offline manner.
* At run time:
  - Does nothing

## Integration

The Pip Install CNB provides `site-packages` as a dependency. Downstream
buildpacks can require the `site-packages` dependency by generating a [Build
Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the dependency provided by the Pip Install Buildpack is
  # "site-packages". This value is considered part of the public API for the
  # buildpack and will not change without a plan for deprecation.
  name = "site-packages"

  # The Pip Install buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the site-packages
    # dependency is available on the $PYTHONPATH for subsequent
    # buildpacks during their build phase. If you are writing a buildpack that
    # needs site-packages during its build process, this flag should be
    # set to true.
    build = true

    # Setting the launch flag to true will ensure that the site-packages
    # dependency is available on the $PYTHONPATH for the running
    # application. If you are writing an application that needs site-packages
    # at runtime, this flag should be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:
```
$ ./scripts/package.sh --version x.x.x
```
This will create a `buildpackage.cnb` file under the build directory which you
can use to build your app as follows: `pack build <app-name> -p <path-to-app>
-b <cpython buildpack> -b <pip buildpack> -b build/buildpackage.cnb -b
<other-buildpacks..>`.

To run the unit and integration tests for this buildpack:
```
$ ./scripts/unit.sh && ./scripts/integration.sh
```

## Configuration

### `BP_PIP_DEST_PATH`

The `BP_PIP_DEST_PATH` variable allows you to specify a custom vendor directory.
This should be a directory underneath the working directory.
Will use `./vendor` if not provided.

```shell
BP_PIP_DEST_PATH=my/custom/vendor-dir
```
