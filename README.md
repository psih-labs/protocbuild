##### protocbuild.yaml
```
root: myprotos
output: gen
protoc_docker_image: registry.gitlab.com/imerkle/grpckit:latest
git:
  org: prototestimerkle
  reporoot: repos
  host: gitlab.com
  branch: master
sources:
  - name: proto
    languages:
      - gogo
default_lang:
  - name: gogo
    args: "paths=source_relative:"
  - name: rust
```