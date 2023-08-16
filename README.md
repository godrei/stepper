# Stepper

Solves some Bitrise step / steplib related tasks.

## stepChanges

Collects step changes from the given time to now in markdown ready format.

## stepLatests

Creates a steps/const.go file for bitrise-init tool with the current latest step versions.

## bitriseSteps

List steps from the Bitrise StepLib.

Examples:

1, List Bitrise maintained Step repositories in 'owner/repo_name' format

```shell
stepper steps \
  --repo-url-filter "https://github.com/bitrise-steplib,https://github.com/bitrise-io" \
  --print-template $'{{range $i, $step := .}}{{$step.Repository.Owner}}/{{$step.Repository.Repo}}\n{{end}}'
```

*NOTE: For the `--print-template` flag, the `$''` syntax is needed because of the `\n` inside of the string.*