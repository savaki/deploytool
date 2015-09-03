package main

import (
	"os"

	"fmt"

	"bitbucket.org/dataskoop/x/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/codegangsta/cli"
)

type Options struct {
	Region  string
	App     string
	Env     string
	Exclude string
	Dry     bool
}

func Opts(c *cli.Context) Options {
	opts := Options{
		Region:  c.String("region"),
		App:     c.String("app"),
		Env:     c.String("env"),
		Exclude: c.String("exclude"),
		Dry:     c.Bool("dry"),
	}

	if opts.App == "" {
		log.Fatalln("ERROR - app flag not specified")
	}
	if opts.Env == "" {
		log.Fatalln("ERROR - env flag not specified")
	}
	if opts.Exclude == "" {
		log.Fatalln("ERROR - exclude flag not specified")
	}

	return opts
}

func main() {
	commonFlags := []cli.Flag{
		cli.StringFlag{"region", "", "default aws region", "AWS_REGION"},
		cli.StringFlag{"app", "", "filter by instances with this app tag", "TEARDOWN_APP"},
		cli.StringFlag{"env", "", "filter by instances with this env tag", "TEARDOWN_ENV"},
		cli.StringFlag{"exclude", "", "filter by instances WITHOUT this version tag", "TEARDOWN_VERSION"},
		cli.BoolFlag{"dry", "dry run, no instances will be terminated", "TEARDOWN_DRY"},
	}

	app := cli.NewApp()
	app.Usage = "terminate ec2 and asg instances"
	app.Commands = []cli.Command{
		{
			Name:   "ec2",
			Usage:  "terminates the specified ec2 instance",
			Action: teardownEC2,
			Flags:  commonFlags,
		},
		{
			Name:   "asg",
			Usage:  "gracefully terminates the specified asg",
			Action: teardownASG,
			Flags:  commonFlags,
		},
	}
	app.Run(os.Args)
}

func teardownEC2(c *cli.Context) {
	opts := Opts(c)

	api := ec2.New(&aws.Config{Region: aws.String(opts.Region)})

	// 1. find instances that match our app/env tags

	output, err := api.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:app"),
				Values: []*string{aws.String(opts.App)},
			},
			{
				Name: aws.String("tag:env"),
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
		log.Printf("DRY RUN - would otherwise terminate %d instance(s)\n", len(instanceIds))
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

func teardownASG(c *cli.Context) {
	opts := Opts(c)
	api := autoscaling.New(&aws.Config{Region: aws.String(opts.Region)})

	fmt.Println(api.Config)
}

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
