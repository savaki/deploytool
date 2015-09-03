package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/codegangsta/cli"
)

func hasEC2Tag(tags []*ec2.Tag, key, value string) bool {
	if tags != nil {
		for _, tag := range tags {
			if *tag.Key == key && *tag.Value == value {
				return true
			}
		}
	}

	return false
}

func teardownEC2(c *cli.Context) {
	opts := Opts(c)

	api := ec2.New(&aws.Config{Region: aws.String(opts.Region)})

	// 1. find instances that match our app/env tags

	output, err := api.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:app"),
				Values: []*string{aws.String(opts.App)},
			},
			{
				Name:   aws.String("tag:env"),
				Values: []*string{aws.String(opts.Env)},
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// 2. construct a list of instanceIds, excluding the version specified

	instanceIds := []*string{}
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			if hasEC2Tag(instance.Tags, "version", opts.Exclude) {
				continue
			}
			instanceIds = append(instanceIds, instance.InstanceId)
		}
	}

	// 3. terminate those instances

	if len(instanceIds) == 0 {
		log.Println("no instances to terminate")
		return
	}

	if opts.Dry {
		log.Printf("dry run mode - would otherwise terminate %d instance(s)\n", len(instanceIds))
		return
	}

	terminateOutput, err := api.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("terminating %d instances", len(terminateOutput.TerminatingInstances))
}
