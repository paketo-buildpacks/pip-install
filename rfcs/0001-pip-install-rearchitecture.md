# Pip Install Rearchitecture
## Proposal
The existing pip buildpack should be rewritten and restructured to only provide the pip dependency. The pip install logic should be factored out into it's own buildpack.

## Motivation
In keeping with the overarching Python Buildpack Rearchitecture RFC, the Pip Install Buildpack should perform one task, which is installing from requirements files. This is part of the effort in Paketo Buildpacks to reduce the responsibilities of each buildpack to make them easier to understand and maintain.

## Implementation
### API
- pip-install
  - requires: cpython and pip during build 
  - provides: site-packages

### Detect
The pip-install buildpack should only detect if there is a `requirements.txt` file at the root of the app.

### Build
#### Configuration
There will be two layers, packages layer and cache layer. 
The packages layer will contain the result of the pip install command.
The cache layer will hold the pip [cache](https://pip.pypa.io/en/stable/reference/pip_install/#caching).

During the build process, the resulting build command will be:
```bash
python -m pip install
  -r <requirements file>                      # install from given requirements file 
  --ignore-installed                          # ignores previously installed packages
  --exists-action=w                           # if path already exists, wipe before installation
  --cache-dir=<path to cache layer directory> # reuse pip cache
  --compile                                   # compile python source files to bytecode
  --user                                      # install to python user install directory set by PYTHONUSERBASE
  --disable-pip-version-check                 # ignore version check warning
```
Upgrade options are ignored if using `--ignore-installed` See [upgrade options](https://pip.pypa.io/en/stable/development/architecture/upgrade-options/).

This should be run with the environment variable `PYTHONUSERBASE` set to the packages layer directory.

If the app has a vendor directory at the root, the app will be considered to be vendored and the resulting build command will be:
```bash
python -m pip install
  -r <requirements file>
  --ignore-installed
  --exists-action=w
  --no-index                                     # ignore package index, uses --find-links URLs 
  --find-links=<file://<vendor layer directory>> # uses apps vendor directory
  --compile
  --user
  --disable-pip-version-check
```
