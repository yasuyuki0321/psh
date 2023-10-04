package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func createTargetList(tagKey, tagValue string) ([]string, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	svc := ec2.NewFromConfig(cfg)
	tagFilter := "tag:" + tagKey

	resp, err := svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   &tagFilter,
				Values: []string{tagValue},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe instances, %v", err)
	}

	targetList := []string{}
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			if instance.PublicIpAddress != nil {
				targetList = append(targetList, *instance.PublicIpAddress)
			}
		}
	}

	return targetList, nil
}
