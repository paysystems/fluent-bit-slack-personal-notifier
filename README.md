# Fluent Bit Slack Personal Notifier

This repository contains the source code for Fluent Bit plugins located in the `plugins` directory.

## Slack Personal Notifier

An *Output Plugin* that sends messages to users specified in the message body.

### Example Configuration:

```ini
[OUTPUT]
    Name     slack_personal_notifier
    Match    *
    Token    xoxb-XXXXXXXXXXX-XXXXXXXXXXXXX-XXXXXXXXXXXXXXXXXXXXXXXX
    Users    {"anton": "UAAAAAAAAA", "leonid": "UBBBBBBBBB"}
    User_Key user
```

### How It Works:

When an input message like `{"user": "leonid", "payload": "text message"}` is received, the plugin will send the `payload` ("text message") to the Slack user with the ID `UBBBBBBBBB`.

## Build Process

The plugins can be built with a single command:

```bash
docker-compose up --build
```

The compiled plugins will be located in the `release` directory.
