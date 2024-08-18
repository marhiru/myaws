package myaws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// EC2RILsOptions customize the behavior of the Ls command.
type EC2RILsOptions struct {
	All    bool
	Quiet  bool
	Fields []string
}

// EC2RILs describes EC2 Reserved Instances.
func (client *Client) EC2RILs(options EC2RILsOptions) error {
	instances, err := client.FindEC2ReservedInstances(options.All)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		fmt.Fprintln(client.stdout, formatEC2RIInstance(client, options, instance))
	}

	return nil
}

func formatEC2RIInstance(client *Client, options EC2RILsOptions, instance *ec2.ReservedInstances) string {
	formatFuncs := map[string]func(client *Client, options EC2RILsOptions, instance *ec2.ReservedInstances) string{
		"ReservedInstancesId": formatEC2ReservedInstanceID,
		"AvailabilityZone":    formatEC2RIAvailabilityZone,
		"InstanceType":        formatEC2RIInstanceType,
		"InstanceCount":       formatEC2RIInstanceCount,
		"State":               formatEC2RIState,
		"Scope":               formatEC2RIScope,
		"Start":               formatEC2RIStart,
		"End":                 formatEC2RIEnd,
		"Duration":            formatEC2RIDuration,
	}

	var outputFields []string
	if options.Quiet {
		outputFields = []string{"InstanceId"}
	} else {
		outputFields = options.Fields
	}

	output := []string{}

	for _, field := range outputFields {
		value := formatFuncs[field](client, options, instance)
		output = append(output, value)
	}

	return strings.Join(output[:], "\t")
}

func formatEC2ReservedInstanceID(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return *instance.ReservedInstancesId
}

func formatEC2RIAvailabilityZone(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	if instance.AvailabilityZone != nil {
		return *instance.AvailabilityZone
	}
	return "N/A"
}

func formatEC2RIInstanceType(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return *instance.InstanceType
}

func formatEC2RIInstanceCount(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return fmt.Sprintf("%3d", *instance.InstanceCount)
}

func formatEC2RIState(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return *instance.State
}

func formatEC2RIScope(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return *instance.Scope
}

func formatEC2RIStart(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return instance.Start.Format("2006-01-02")
}

func formatEC2RIEnd(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return instance.End.Format("2006-01-02")
}

func formatEC2RIDuration(_ *Client, _ EC2RILsOptions, instance *ec2.ReservedInstances) string {
	return fmt.Sprintf("%2dyear", *instance.Duration/(3600*24*365))
}
