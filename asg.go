package main

import (
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/codegangsta/cli"
)

func teardownASG(c *cli.Context) {
	opts := Opts(c)
	api := autoscaling.New(&aws.Config{Region: aws.String(opts.Region)})

	instances, err := findASGInstancesToTerminate(api, opts)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("terminating %d autoscaling group(s)\n", len(instances))

	if opts.Dry {
		log.Println("dry run mode - no autoscaling groups will be terminages\n")
		return
	}

	wg := &sync.WaitGroup{}
	for _, instance := range instances {
		wg.Add(1)
		go terminateASG(api, instance, wg)
	}
	wg.Wait()
}

func terminateASG(api *autoscaling.AutoScaling, instance *autoscaling.Group, wg *sync.WaitGroup) error {
	defer wg.Done()

	// 1. set the number of instances to 0
	_, err := api.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: instance.AutoScalingGroupName,
		DesiredCapacity:      aws.Int64(0),
		MaxSize:              aws.Int64(0),
		MinSize:              aws.Int64(0),
	})
	if err != nil {
		return err
	}

	// 2. wait until the instance count is 0
	for {
		instances, err := api.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{instance.AutoScalingGroupName},
		})
		if err != nil {
			return err
		}

		// instance already deleted?  we're done
		if len(instances.AutoScalingGroups) == 0 {
			return nil
		}

		// no instances remaining in asg?  break
		if v := instances.AutoScalingGroups[0]; len(v.Instances) == 0 {
			break
		}

		time.Sleep(15 * time.Second)
	}

	// 3. terminate the asg
	log.Printf("deleting autoscaling group, %s\n", *instance.AutoScalingGroupName)
	_, err = api.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: instance.AutoScalingGroupName,
		ForceDelete:          aws.Bool(true),
	})
	if err != nil {
		return err
	}

	log.Printf("deleting launch configuration, %s\n", *instance.LaunchConfigurationName)
	_, err = api.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: instance.LaunchConfigurationName,
	})
	if err != nil {
		return err
	}

	return nil
}

func findASGInstancesToTerminate(api *autoscaling.AutoScaling, opts Options) ([]*autoscaling.Group, error) {
	groups, err := api.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, err
	}

	instances := []*autoscaling.Group{}
	for _, asg := range groups.AutoScalingGroups {
		if !hasASGTags(asg, "app", opts.App) {
			continue
		}
		if !hasASGTags(asg, "env", opts.Env) {
			continue
		}
		if hasASGTags(asg, "version", opts.Exclude) {
			continue
		}

		instances = append(instances, asg)
	}

	return instances, nil
}

func hasASGTags(asg *autoscaling.Group, key, value string) bool {
	for _, tag := range asg.Tags {
		if *tag.Key == key && *tag.Value == value {
			return true
		}
	}

	return false
}
