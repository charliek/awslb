# awslb
Write out a proxy configuration based on polling the aws api

This application can be used to create configuration files for
load balancing applications like haproxy or nginx. It is able to
either lookup instances associated with an ELB or use an EC2
search with multiple filters to locate instances. The configuration
for this looks like:

```yaml
# How frequent you want to poll aws in seconds
polling_seconds: 180

# The location of the go template used to build the configuration file
source: /etc/awslb/template.cfg

# The location of the configuration file to write out
destination: /etc/haproxy/haproxy.cfg

# The command to run when the configuration file has changed
command: /bin/reload_haproxy

# This list can be one or more services. Currently the only types
# supported are "search" and "elb". For now consider all fields
# required.
services:

  # Name of the application which will be used in the template later
  application1:
    # Discovery type (elb or search)
    type: search
    search:
      # The region to make the api call in
      region: us-east-1

      # The filters to use in the search. The filters are pipe delimited
      # and whitespace is trimmed. The first value is the filter name and
      # values after that are the permitted values. Multiple values are
      # allowed.
      # http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstances.html
      filters:
        - "vpc-id | vpc-094b4701"
        - "tag:app_name | application1"

  # name of the application which will be used in the template later
  application2:
    # discovery type (elb or search)
    type: elb
    elb:
      # The name of the ELB to inspect the instances of
      name: dev-application2-lb

      # Should unhealthy instances be reported
      check_health: false

      # The region to look up instances in
      region: us-east-1
```

Once this is in place and the application is started it is required to have
a go template that looks something like:

```
...

listen devconn
    bind *:4040
        mode tcp
        balance leastconn {{range lookupService "application1"}} {{if .PrivateIP}}
        server {{.ID}} {{.PrivateIP}}:9090 check{{end}}{{end}}
...
```

This is a Go template which has syntax documented online at:
http://golang.org/pkg/text/template/

The interesting part here is the lookupService call. The string passed to this call
must match one of the service names described in the configuration. It will then return
a list of instances that will be looped over and written out.  Each instance object has
the values `ID`, `PrivateIP`, and `PublicIP`. Note that the IP variables could potentially
be null.

The application can be started via the command line like:

```bash
awslb /etc/awslb/lb.yml
```

Once started it will immediately try to create the configured configuration file and
will then call the configured command which should notify your proxy to reload. Then
every configured amount of time it will call amazon and if the output of the configuration
file changes the command will be called again.


