package main

import (
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

// Instance is the structured that the configuration writer expects to be
// delivered by the providers
type Instance struct {
	ID        string
	PublicIP  *string
	PrivateIP *string
}

func logAwsError(err error) {
	if awserr, ok := err.(awserr.Error); ok {
		log.Error("AWS returned an error: ", awserr.Code(), awserr.Message())
	} else {
		log.Error("Unknown error occurred: ", err)
	}
}

func buildInstanceList(resp *ec2.DescribeInstancesOutput) []*Instance {
	instances := []*Instance{}
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			instance := &Instance{
				ID:        *i.InstanceID,
				PublicIP:  i.PublicIPAddress,
				PrivateIP: i.PrivateIPAddress,
			}
			instances = append(instances, instance)
		}
	}
	return instances
}

// NewSearchLookup creates the structure needed to lookup instances
// based on a aws search.
func NewSearchLookup(config SearchConfig) *SearchLookup {
	return &SearchLookup{
		config:      config,
		ec2Client:   ec2.New(&aws.Config{Region: config.Region}),
		searchInput: &ec2.DescribeInstancesInput{Filters: createFilters(config.Filters)},
	}
}

// SearchLookup will lookup instances based on the provided search
type SearchLookup struct {
	config      SearchConfig
	ec2Client   *ec2.EC2
	searchInput *ec2.DescribeInstancesInput
}

func createFilter(s string) *ec2.Filter {
	parts := strings.Split(s, "|")
	if len(parts) < 2 {
		log.Warnf("Filter of '%s' is not valid", s)
		return nil
	}
	values := make([]*string, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		trimmed := strings.TrimSpace(parts[i])
		values[i-1] = &trimmed
	}
	trimmedName := strings.TrimSpace(parts[0])
	return &ec2.Filter{
		Name:   &trimmedName,
		Values: values,
	}
}

func createFilters(sFilters []string) []*ec2.Filter {
	filters := []*ec2.Filter{}
	for _, s := range sFilters {
		filter := createFilter(s)
		if filter != nil {
			filters = append(filters, filter)
		}
	}
	return filters
}

// Lookup will execute an aws api call to search for the provided instances
func (l *SearchLookup) Lookup() ([]*Instance, error) {
	resp, err := l.ec2Client.DescribeInstances(l.searchInput)
	if err != nil {
		logAwsError(err)
		return nil, err
	}
	return buildInstanceList(resp), nil
}

// NewElbLookup will create the structure needed to do aws searches for
// instances attached to the configured elb
func NewElbLookup(config ElbConfig) *ElbLookup {
	ec2Srv := ec2.New(&aws.Config{Region: config.Region})
	elbSrv := elb.New(&aws.Config{Region: config.Region})
	return &ElbLookup{
		config:    config,
		ec2Client: ec2Srv,
		elbClient: elbSrv,
	}
}

// ElbLookup will return a list of instances in the given ELB
type ElbLookup struct {
	config    ElbConfig
	ec2Client *ec2.EC2
	elbClient *elb.ELB
}

// Lookup will hit the aws api and find instances attached to the given ELB
func (l *ElbLookup) Lookup() ([]*Instance, error) {
	output, err := l.elbClient.DescribeInstanceHealth(&elb.DescribeInstanceHealthInput{LoadBalancerName: &l.config.Name})
	if err != nil {
		logAwsError(err)
		return nil, err
	}

	healthyIds := []*string{}
	for _, state := range output.InstanceStates {
		if !l.config.CheckHealth || *state.State == "InService" {
			healthyIds = append(healthyIds, state.InstanceID)
		}
	}

	resp, err := l.ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIDs: healthyIds})
	if err != nil {
		logAwsError(err)
		return nil, err
	}
	return buildInstanceList(resp), nil
}
