#!/usr/bin/env bash

bazel run //:bazeldnf -- fetch \
  --repofile rpm/repo.yaml
bazel run //:bazeldnf -- rpmtree \
  --workspace rpm/rpms.bzl \
  --buildfile rpm/BUILD.bazel \
  --repofile rpm/repo.yaml \
  --name cryptsetup-devel \
  cryptsetup-devel
bazel run //:bazeldnf -- rpmtree \
  --workspace rpm/rpms.bzl \
  --buildfile rpm/BUILD.bazel \
  --repofile rpm/repo.yaml \
  --name libvirt-devel \
  libvirt-devel
bazel run //rpm:ldd-cryptsetup
bazel run //rpm:ldd-libvirt
bazel run //rpm:ldd-glibc
