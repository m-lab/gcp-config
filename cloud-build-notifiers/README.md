## Cloud Build notifiers

The files in this directory are used to configure Cloud Build notifiers.
Currently, the only notifier enabled is Slack. See these documents for more
infomation on how the Slack notifier was enabled in each project.

* https://cloud.google.com/build/docs/configuring-notifications/configure-slack
* https://cloud.google.com/build/docs/configuring-notifications/automate

There are two configuration files for the Slack notifier in this repo:

* slack-configuration.yaml

This is the configuration file passed to the Slack Cloud Build notifier, which
is a container image that gets launched by Cloud Run when it receives a Pub/Sub
push notification that a Cloud Build failed. The file contains two
`{{PROJECT}}` template variables which need to be substituted before using.

* slack-message-template.json

This is the Slack message template. It uses the [Slack Block
Kit](https://api.slack.com/block-kit) for generating pretty Slack messages. Of
note is that it does not contain the typical `blocks: []` field that the Block Kit
generally wants to see defined in the JSON. Instead, the content should only be
the array that is assigned to `blocks:`.

