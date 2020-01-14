# Design of Cloud Build Conditions (cbif)

"cbif" is an alternate Docker image entrypoint for creating conditional
execution in the [Google Cloud Build][1] environment.

[1]: https://cloud.google.com/cloud-build/docs/build-config

## Rationale

cbif is designed to complement the existing functionality of Cloud Build by
providing a mechanism for conditional actions. We created cbif to:

* consolidate configurations for test, build, and deployment.
* keep test and deployment steps "out in the open" with respect to one another.
* reduce ad-hoc, manual overhead for managing these configurations.

For alternatives considered, see further below.

As an entrypoint, cbif is the first command to run. This means that cbif can
read configuration from environment variables, optionally prepare the local
environment (e.g. symlink workspace, checkout .git directory), before running
user-provided commands.

## Requirements

* cbif should be easy to include in derived images.
* cbif should be configured through environment to keep commands separate.
* cbif should be language independent (as much as possible).

## Supported cbif Options

Cloud Build provides ["substitutions"][2] which allow injection of
build-specific environment variables. However, they are not part of the Linux
environment unless specified explicitly.

[2]: https://cloud.google.com/cloud-build/docs/configuring-builds/substitute-variable-values

cbif takes the logical AND of all assigned, conditional directives.

```yaml
- name: cbif
  env:
  # CONDITION: Run commands if the current $PROJECT_ID is one of the named
  # projects. Default to all projects.
  - PROJECT_IN=proj1[,proj2,...]

  # CONDITION: Run commands if the current $BRANCH_NAME is one of the named
  # branches. Default to all branches.
  - BRANCH_IN=branch1[,branch2,...]

  # CONDITION: Run commands if the current $TAG_NAME value is not empty.
  - TAG_IS_DEFINED=<value>

  # CONDITION: Run commands if the current $_PR_NUMBER value is not empty.
  - PR_IS_DEFINED=<value>

  # EXECUTION: Continue running commands even if one returns an error.
  # Default false.
  - IGNORE_ERRORS=bool

  # EXECUTION: Treat cbif arguments as parameters to a single command rather
  # than treating each parameter as independent commands.
  # Default false.
  - SINGLE_COMMAND=bool

  # EXECUTION: Limit the time each command runs to the given timeout.
  # Default 1h.
  - COMMAND_TIMEOUT=duration

  # SETUP: Link workspace as path will create a sylink at the named path that
  # links to the named WORKSPACE and then change the PWD to that directory
  # before executing commands. No default.
  - WORKSPACE_LINK=<path>

  # SETUP: The upstream git URL suitable for using with `git clone`.
  # Credentials not yet supported. No default.
  - GIT_ORIGIN_URL=<url>

  # SETUP: Git commit sha of the current build.
  # No default.
  - COMMIT_SHA=<value>

  # SETUP: The directory to target when using the WORKSPACE_LINK option.
  # Default /workspace.
  - WORKSPACE=<path>
  args:
  - cmd1 [arg1 ... argN]
  - ...
  - cmdN [arg1 ... argN]
```

## Alternatives Considered

* Why not use a Dockerfile to run tests?

  That may be a good option. This requires a Dockerfile to setup a test
  environment, a step in CB to build it, and a step to run a specific script
  in the new image to run the test. Changes to the test would require updates
  to the Dockerfile and the test script. How the test is called is defined in
  the cloudbuild.yaml and the actual steps are in separate files.

  The goal of cbif is to move more of the test script logic into
  cloudbuild.yaml. This way, the environment build and use of that
  environment are in the same place. The simpler the test environment is, the
  better. This model is similar to other CI environments like Travis-CI, or
  Circle-CI, and others.

* Why not use multiple Cloud Build configuration files?

  This may be a good option. CB can trigger on branch pushes, pull requests,
  and tags. Some users manage deployments across multiple GCP projects. Each
  trigger type may require steps that should be omitted from other triggers,
  e.g. deploy to a testing project for push events, only run tests for pull
  requests.

  The goal of cbif is to allow more of the configuration to go into a single
  config file, reducing redundancy and making inter-step dependencies clearer.

## Questions

* Can you put secrets from multiple projects in the same CB file?
