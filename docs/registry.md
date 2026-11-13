# Container registry

Lyra serves the OCI distribution API at `/v2/` on any host it answers, using the embedded
`distribution` registry. An image is named after a Lyra repository (`owner/repo`),
optionally with more path components: `registry.example.com/krosf/lyra:v1` or
`registry.example.com/krosf/lyra/runner:v1`. Pushing to a name whose repository doesn't
exist is refused with `NAME_UNKNOWN`.

```sh
docker login registry.example.com -u lyra --password-stdin <<< "$LYRA_TOKEN"
docker push registry.example.com/krosf/lyra:v1
lyra image list <repo>
lyra image delete <name>:<tag>|<name>@<digest> --yes
```

`--registry-host` sets the host name shown in pull references; without it, the request's
host.

- **Access:** pulling needs `read` on the repository, pushing `write`, deleting `admin`.
  Mounting a layer from another repository needs `read` there; without it, the client
  uploads the layer instead. The catalog (`/v2/_catalog`) needs an unrestricted token.
- **Login:** behind the Tailscale proxy, clients on the tailnet need no login. Otherwise any
  user name works, with a token as the password.
- **Tracking:** each tag pushed is recorded with its digest and pusher, as `image.pushed`;
  deletions as `image.deleted`. Manifests pushed only by digest, such as the per-platform
  manifests of a multi-arch image, aren't listed.
- **Deleting:** `lyra image delete` removes a tag, or a manifest and every tag pointing at
  it. Deleting a repository deletes its images' tags and manifests too.
- **Freeing space:** `lyra admin registry-gc` deletes layers and manifests no tag refers to
  (directly or through a multi-arch index), including those of deleted repositories. Lyra
  is read-only while it runs.
