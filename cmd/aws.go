package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type InstanceInfo struct {
	IP   string
	Name string
}

const (
	EC2RunningStateCode = 16
)

func createServiceClient() (svc *ec2.Client, err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}
	svc = ec2.NewFromConfig(cfg)

	return svc, nil
}

func describeInstances(svc *ec2.Client, tags map[string]string) (resp *ec2.DescribeInstancesOutput, err error) {
	var filters []types.Filter

	for key, value := range tags {
		tagFilter := "tag:" + key
		filters = append(filters, types.Filter{
			Name:   &tagFilter,
			Values: []string{value},
		})
	}

	return svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: filters,
	})
}

func createTargetList(tags map[string]string, ipType string) (map[string]InstanceInfo, error) {
	svc, err := createServiceClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create service client, %v", err)
	}

	resp, err := describeInstances(svc, tags)
	if err != nil {
		return nil, fmt.Errorf("unable to describe instances, %v", err)
	}

	targetList, err := extractTargets(resp, ipType)
	if err != nil {
		return nil, fmt.Errorf("unable to extract targets, %v", err)
	}

	if len(targetList) == 0 {
		return nil, fmt.Errorf("no targets found")
	}

	return targetList, nil
}

func extractTargets(resp *ec2.DescribeInstancesOutput, ipType string) (map[string]InstanceInfo, error) {
	targetList := map[string]InstanceInfo{}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			if *instance.State.Code == EC2RunningStateCode {
				var ip string
				switch ipType {
				case "public":
					if instance.PublicIpAddress != nil {
						ip = *instance.PublicIpAddress
					}
				case "private":
					if instance.PrivateIpAddress != nil {
						ip = *instance.PrivateIpAddress
					}
				default:
					return nil, fmt.Errorf("ipType is invalid: %v", ipType)
				}

				name := ""
				for _, tag := range instance.Tags {
					if *tag.Key == "Name" {
						name = *tag.Value
						break
					}
				}
				if name == "" {
					name = "-"
				}
				targetList[*instance.InstanceId] = InstanceInfo{IP: ip, Name: name}
			}
		}
	}
	return targetList, nil
}
