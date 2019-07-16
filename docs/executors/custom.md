# The Custom executor

> [Introduced](https://gitlab.com/gitlab-org/gitlab-runner/issues/2885) in GitLab Runner 12.1

GitLab Runner provides the Custom executor for environments that it
doesn't support natively, for example, Podman or Libvirt.

This gives you the control to create your own executor by configuring
GitLab Runner to use some executable to provision, run, and clean up
your environment.

## Limitations

Below are some current limitations when using the Custom executor:

- No support for [`image`](https://docs.gitlab.com/ee/ci/yaml/#image).
  See [#4357](https://gitlab.com/gitlab-org/gitlab-runner/issues/4357)
  for more details.
- No support for
  [`services`](https://docs.gitlab.com/ee/ci/yaml/#services). See
  [#4358](https://gitlab.com/gitlab-org/gitlab-runner/issues/4358) for
  more details.
- No [Interactive Web
  Terminal](https://docs.gitlab.com/ee/ci/interactive_web_terminal/) support.

## Configuration

There are a few configuration keys that you can choose from. Some of them are optional.

Below is an example of configuration for the Custom executor using all available
configuration keys:

```toml
[[runners]]
  name = "custom"
  url = "https://gitlab.com"
  token = "TOKEN"
  executor = "custom"
  builds_dir = "/builds"
  cache_dir = "/cache"
  [runners.custom]
    config_exec = "/path/to/config.sh"
    config_args = [ "SomeArg" ]
    config_exec_timeout = 200

    prepare_exec = "/path/to/script.sh"
    prepare_args = [ "SomeArg" ]
    prepare_exec_timeout = 200

    run_exec = "/path/to/binary"
    run_args = [ "SomeArg" ]

    cleanup_exec = "/path/to/executable"
    cleanup_args = [ "SomeArg" ]
    cleanup_exec_timeout = 200

    graceful_kill_timeout = 200
    force_kill_timeout = 200
```

For field definitions and which ones are required, see
[`[runners.custom]`
section](../configuration/advanced-configuration.md#the-runnerscustom-section)
configuration.

In addition both `builds_dir` and `cache_dir` inside of the
[`[[runners]]`](../configuration/advanced-configuration.md#the-runners-section)
are required fields.

## Prerequisite software for running a Job

The user must set up the environment, including the following that must
be present in the `PATH`:

- [git](https://git-scm.com/download): Used to clone the repositories.
- [git-lfs](https://git-lfs.github.com/): Pulls any LFS objects that
  might be in the repository.
- [gitlab-runner](https://docs.gitlab.com/runner/install/): Used to
  download/update artifacts and cache.

## Stages

The Custom executor provides the stages for you to configure some
details of the job, prepare and cleanup the environment and run the job
script within it. Each stage is responsible for specific things and has
different things to keep in mind.

Each stage executed by the Customer executor is executed at the time
a builtin GitLab Runner executor would execute them.

For each step that will be executed, specific environment variables are
exposed to the executable, which can be used to get information about
the specific Job that is running. All stages will have the following
environment variables available to them:

- Standard CI/CD [environment
  variables](https://docs.gitlab.com/ee/ci/variables/), including
  [predefined
  variables](https://docs.gitlab.com/ee/ci/variables/predefined_variables.html).
- All environment variables provided by the Customer Runner host system.

Both CI/CD environment variables and predefined variables are prefixed
with `CUSTOM_ENV_` to prevent conflicts with system environment
variables. For example, `CI_BUILDS_DIR` will be available as
`CUSTOM_ENV_CI_BUILDS_DIR`.

The stages run in the following sequence:

1. `config_exec`
1. `prepare_exec`
1. `run_exec`
1. `cleanup_exec`

### Config

The Config stage is executed by `config_exec`.

Sometimes you might want to set some settings during execution time. For
example settings a build directory depending on the project ID.
`config_exec` reads from STDOUT and expects a valid JSON string with
specific keys.

For example:

```sh
#!/usr/bin/env bash

cat << EOS
{
  "builds_dir": "/builds/${CUSTOM_ENV_CI_CONCURRENT_PROJECT_ID}/${CUSTOM_ENV_CI_PROJECT_PATH_SLUG}"
  "cache_dir": "/cache/${CUSTOM_ENV_CI_CONCURRENT_PROJECT_ID}/${CUSTOM_ENV_CI_PROJECT_PATH_SLUG}"
  "builds_dir_is_shared": true
}
EOS
```

Any additional keys inside of the JSON string will be ignored. If it's
not a valid JSON string the stage will fail and be retried two more
times.

| Parameter | Type | Required | Allowed empty | Description |
|-----------|------|----------|---------------|-------------|
| `builds_dir` | string | ✗ | ✗ | The base directory where the working directory of the job will be created. |
| `cache_dir` | string | ✗ | ✗ | The base directory where local cache will be stored. |
| `builds_dir_is_shared` | bool | ✗ | n/a | Defines whether the environment is shared between concurrent job or not. |

The `STDERR` of the executable will print to the job log.

The user can set
[`config_exec_timeout`](../configuration/advanced-configuration.md#the-runnerscustom-section)
if they want to set a deadline for how long GitLab Runner should wait to
return the JSON string before terminating the process.

If any of the
[`config_exec_args`](../configuration/advanced-configuration.md#the-runnerscustom-section)
are defined, these will be added in order to the executable defined in
`config_exec`. For example we have the `config.toml` content below:

```toml
...
[runners.custom]
  ...
  config_exec = "/path/to/config"
  config_args = [ "Arg1", "Arg2" ]
  ...
```

GitLab Runner would execute it as `/path/to/config Arg1 Arg2`.

### Prepare

The Prepare stage is executed by `prepare_exec`.

At this point, GitLab Runner knows everything about the job (where and
how it's going to run). The only thing left is for the environment to be
set up so the job can run.  GitLab Runner will execute the executable
that is specified in `prepare_exec`.

This is responsible for setting up the environment (for example,
creating the virtual machine or container, or anything else). After this
is done, we expect that the environment is ready to run the job.

This stage is executed only once, in a job execution.

The user can set
[`prepare_exec_timeout`](../configuration/advanced-configuration.md#the-runnerscustom-section)
if they want to set a deadline for how long GitLab Runner
should wait to prepare the environment before terminating the process.

The `STDOUT` and `STDERR` returned from this executable will print to
the job log.

If any of the
[`prepare_exec_args`](../configuration/advanced-configuration.md#the-runnerscustom-section)
are defined, these will be added in order to the executable defined in
`prepare_exec`. For example we have the `config.toml` content below:

```toml
...
[runners.custom]
  ...
  preapre_exec = "/path/to/bin"
  prepare_args = [ "Arg1", "Arg2" ]
  ...
```

GitLab Runner would execute it as `/path/to/bin Arg1 Arg2`.

### Run

The Run stage is executed by `run_exec`.

The `STDOUT` and `STDERR` returned from this executable will print to
the job log.

Unlike the other stages, the `run_exec` stage is executed multiple
times, since it's split into sub stages listed below in sequential
order:

1. `prepare_script`
1. `get_sources`
1. `restore_cache`
1. `download_artifacts`
1. `build_script`
1. `after_script`
1. `archive_cache`
1. `upload_artifact_on_success` OR `upload_artifact_on_failure`

For each stage mentioned above, the `run_exec` executable will be
executed with:

- The usual environment variables.
- Two arguments:
  - The path to the script that GitLab Runner creates for the Custom
    executor to run.
  - Name of the stage.

For example:

```sh
/path/to/run_exec.sh /path/to/tmp/script1 prepare_executor
/path/to/run_exec.sh /path/to/tmp/script1 prepare_script
/path/to/run_exec.sh /path/to/tmp/script1 get_sources
```

If you have `run_args` defined, they are the first set of arguments
passed to the `run_exec` executable, then GitLab Runner adds others. For
example, suppose we have the following `config.toml`:

```toml
...
[runners.custom]
  ...
  run_exec = "/path/to/run_exec.sh"
  run_args = [ "Arg1", "Arg2" ]
  ...
```

GitLab Runner will execute the executable with the following arguments:

```sh
/path/to/run_exec.sh Arg1 Arg2 /path/to/tmp/script1 prepare_executor
/path/to/run_exec.sh Arg1 Arg2 /path/to/tmp/script1 prepare_script
/path/to/run_exec.sh Arg1 Arg2 /path/to/tmp/script1 get_sources
```

This executable should be responsible of executing the scripts that are
specified in the first argument. They contain all the scripts any GitLab
Runner executor would run normally to clone, download artifacts, run
user scripts and all the other steps described below. The scripts can be
of the following shells:

- Bash
- PowerShell
- Batch (Deprecated)

We generate the script using the shell configured by `shell` inside of
[`[[runners]]`](../configuration/advanced-configuration.md#the-runners-section).
If none is provided the defaults for the OS platform are used.

The table below is a detailed explanation of what each script does and
what the main goal of that script is.

| Script Name | Script Contents |
|:-----------:|:---------------:|
| `prepare_script` | Simple debug info on which machine the Job is running on. |
| `get_srouces`    | Prepares the Git config, and clone/fetch the repository. We suggest you keep this as is since you get all of the benefits of git strategies that GitLab provides. |
| `restore_cache` | Extract the cache if any are defined. This expects the `gitlab-runner` binary is available in `$PATH`. |
| `download_artifacts` | Download artifacts, if any are defined. This expects `gitlab-runner` binary is available in `$PATH`. |
| `build_script` | This is a combination of [`before_script`](https://docs.gitlab.com/ee/ci/yaml/#before_script-and-after_script) and [script](https://docs.gitlab.com/ee/ci/yaml/#script). |
| `after_script` | This is the [`after_script`](https://docs.gitlab.com/ee/ci/yaml/#before_script-and-after_script) defined from the job. This is always called even if any of the previous steps failed. |
| `archive_cache` | Will create an archive of all the cache, if any are defined. |
| `upload_artifact_on_success` | Upload any artifacts that are defined. Only executed when `build_script` was successful. |
| `upload_artifact_on_failure` | Upload any artifacts that are defined. Only exected when `build_script` fails. |

### Cleanup

The Cleanup stage is executed by `cleanup_exec`.

This final stage is executed even if one of the previous stages failed.
The main goal for this stage is to clean up any of the environments that
might have been set up. For example, turning off VMs or deleting
containers.

The result of `cleanup_exec` does not affect job statuses. For example,
a job will be marked as successful even if the following occurs:

- Both `prepare_exec` and `run_exec` are successful.
- `cleanup_exec` fails.

The user can set
[`cleanup_exec_timeout`](../configuration/advanced-configuration.md#the-runnerscustom-section)
if they want to set some kind of deadline of how long GitLab Runner
should wait to clean up the environment before terminating the
process.

The `STDOUT` of this executable will be printed to GitLab Runner logs at a
DEBUG level. The `STDERR` will be printed to the logs at a WARN level.

If any of the
[`cleanup_exec_args`](../configuration/advanced-configuration.md#the-runnerscustom-section)
are defined, these will be added in order to the executable defined in
`cleanup_exec`. For example we have the `config.toml` content below:


```toml
...
[runners.custom]
  ...
  cleanup_exec = "/path/to/bin"
  cleanup_args = [ "Arg1", "Arg2" ]
  ...
```

GitLab Runner would execute it as `/path/to/bin Arg1 Arg2`.

## Terminating and killing executables

GitLab Runner will try to gracefully terminate an executable under any
of the following conditions:

- `config_exec_tiemout`, `prepare_exec_timeout` or `cleanup_exec_timeout` are met.
- The job [times out](https://docs.gitlab.com/ee/user/project/pipelines/settings.html#timeout).

When a timeout is reached, a `SIGTERM` is sent to the executable, and
the countdown for
[`exec_terminate_timeout`](../configuration/advanced-configuration.md#the-runnerscustom-section)
starts. The executable should listen to this signal to make sure it
cleans up any resources. If `exec_terminate_timeout` passes and the
process is still running, a `SIGKILL` is sent to kill the process and
[`exec_force_kill_timeout`](../configuration/advanced-configuration.md#the-runnerscustom-section)
will start. If the process is still running after
`exec_force_kill_timeout` has finished, GitLab Runner will abandon the
process and will not try to stop/kill anymore. If both these timeouts
are reached during `config_exec`, `prepare_exec` or `run_exec` the build
is marked as failed.

## Error handling

There are two types of errors that GitLab Runner can handle differently.
These errors are only handled when the executable inside of
`config_exec`, `prepare_exec`, `run_exec`, and `cleanup_exec` exits with
these codes. If the user exits with a non-zero exit code, it should be
propagated as one of the error codes below.

If the user script exits with one of these code it has to
be propagated to the executable exit code.

### Build Failure

GitLab Runner provides `BUILD_FAILURE_EXIT_CODE` environment
variable which should be used by the executable as an exit code to
inform GitLab Runner that there is a failure on the users job. If the
executable exits with the code from
`BUILD_FAILURE_EXIT_CODE`, the build is marked as a failure
appropriately in GitLab CI.

If the script that the user defines inside of `.gitlab-ci.yml` file
exits with a non-zero code, `run_exec` should exit with
`BUILD_FAILURE_EXIT_CODE` value.

NOTE: **Note:**
We strongly suggest using `BUILD_FAILURE_EXIT_CODE` to exit
instead of a hard coded value since it can change in any release, making
your binary/script future proof.

### System Failure

You can send a system failure to GitLab Runner by exiting the process with the
error code specified in the `SYSTEM_FAILURE_EXIT_CODE`. If this error
code is returned, on certain stages GitLab Runner will retry the stage, if none
of the retries are successful the job will be marked as failed.

Below is a table of what stages are retried, and by how many times.

| Stage Name           | Number of attempts                                          | Duration to wait between each retry |
|----------------------|-------------------------------------------------------------|-------------------------------------|
| `preapre_exec`       | 3                                                           | 3 seconds                           |
| `get_sources`        | Value of `GET_SOURCES_ATTEMPTS` variable. (Default 1)       | 0 seconds                           |
| `restore_cache`      | Value of `RESTORE_CACHE_ATTEMPTS` variable. (Default 1)     | 0 seconds                           |
| `download_artifacts` | Value of `ARTIFACT_DOWNLOAD_ATTEMPTS` variable. (Default 1) | 0 seconds                           |

NOTE: **Note:**
We strongly suggest using `SYSTEM_FAILURE_EXIT_CODE` to exit
instead of a hard coded value since it can change in any release, making
your binary/script future proof.
