"""Import the private Redis service lock independently of production APT inputs."""

load("@rules_distroless//apt:extensions.bzl", _apt = "apt")

apt = _apt
