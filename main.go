package main

import (
	"os"

	"bitbucket.org/dataskoop/x/log"
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
