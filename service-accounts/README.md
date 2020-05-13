# Custom Roles

Create custom roles for a project:

```
for file in *.yaml ; do
  echo gcloud --project $PROJECT iam roles create ${file%%.yaml} --file=$file
done
```

