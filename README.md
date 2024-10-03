# pickabot

A bot to pick things, notably pull request reviewers

Owned by eng-infra

## Deploying

```
ark start pickabot -e production
```

## How to Test

You can test pickabot using the `pickabot-dev` bot in Slack.

In Slack, you will navigate to `pickabot-dev` as you would if chatting with any App via DM.

### Starting a Test Instance

To start a test instance of pickabot, and to run tests in Slack, do the following:

1. Check that pickabot-dev is enabled in Slack. You may need to check with IT to ask IT to enable the pickabot-dev bot if it is not already.

2. Start a local instance of pickabot-dev by running `ark start -l` as you would any local service you're testing.

3. Once your local test instance is running, you can send messages by messaging `@pickabot-dev` in Slack. (To verify the name of the dev pickabot, look at: `deployment.yml`)