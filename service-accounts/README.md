# Custom Roles

Create custom roles for a project:

```sh
for file in *.yaml ; do
  echo gcloud --project $PROJECT iam roles create ${file%%.yaml} --file=$file
done
```

## Export exiting custom roles

If you've created a role manually, use the following steps to export the
config. First, find the current role:

```sh
gcloud --project $PROJECT iam roles list
```

Next, describe the role. The output format is YAML.

```sh
gcloud --project $PROJECT iam roles describe appengine_flexible_deployer \
    | grep -v 'etag:|name:' > appengine_flexible_deployer.yaml
```

NOTE: GCP IAM role ids cannot use '-' in their names. So that files and roles
can have the same names (and simplify automated management), please use
underscore '_' as a word separator where helpful.

NOTE: we strip the `etag` and `name` fields so that the configuration can be
applied to other projects easily.
