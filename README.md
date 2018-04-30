# send-pmg-report

Send a customized spam report on Proxmox Mailgateway. Customized means that
you can use this utility to redirect reports to different users, which is very
useful when e.g. dealing with mailing lists, since the reports shouldn't go to
the list...

## Configuration
Disable the reports in PMG (set the report style to 'none') and create
`config.yml` like this:
```
redirectedDomains:
  - domain: lists.example.com
    destination: spammod@example.com
redirectedTargets:
  - target: someimportantuser@example.com
    destination: spammod@example.com
```

## Building
This project uses `dep`, so you'll need to:
```
dep ensure
go build
```

We also include an example systemd configuration to send the reports each
sunday at 23:00.
