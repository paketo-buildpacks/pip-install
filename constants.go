package pipinstall

// SitePackages is the name of the dependency provided by the Pip Install
// buildpack.
const SitePackages = "site-packages"

// CPython is the name of the python runtime dependency provided by the CPython
// buildpack: https://github.com/paketo-buildpacks/cpython.
const CPython = "cpython"

// Pip is the name of the dependency provided by the Pip buildpack:
// https://github.com/paketo-buildpacks/pip.
const Pip = "pip"

// The layer name for packages layer. This layer is where dependencies are
// installed to.
const PackagesLayerName = "packages"

// The layer name for cache layer. This layer holds the pip cache.
const CacheLayerName = "cache"
