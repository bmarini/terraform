package aws

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/autoscaling"
)

func resource_aws_autoscaling_group_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)
	log.Println(rs.Attributes["availability_zones"])

	var err error
	autoScalingGroupOpts := autoscaling.CreateAutoScalingGroup{}

	if rs.Attributes["min_size"] != "" {
		autoScalingGroupOpts.MinSize, err = strconv.Atoi(rs.Attributes["min_size"])
	}

	if rs.Attributes["max_size"] != "" {
		autoScalingGroupOpts.MaxSize, err = strconv.Atoi(rs.Attributes["max_size"])
	}

	if rs.Attributes["default_cooldown"] != "" {
		autoScalingGroupOpts.DefaultCooldown, err = strconv.Atoi(rs.Attributes["default_cooldown"])
	}

	if rs.Attributes["desired_capicity"] != "" {
		autoScalingGroupOpts.DesiredCapacity, err = strconv.Atoi(rs.Attributes["desired_capicity"])
	}

	if rs.Attributes["healthcheck_grace_period"] != "" {
		autoScalingGroupOpts.HealthCheckGracePeriod, err = strconv.Atoi(rs.Attributes["healthcheck_grace_period"])
	}

	if err != nil {
		return nil, fmt.Errorf("Error parsing configuration: %s", err)
	}

	if rs.Attributes["availability_zones"] != "" {
		autoScalingGroupOpts.AvailZone = expandStringList(flatmap.Expand(
			rs.Attributes, "availability_zones").([]interface{}))
	}

	if rs.Attributes["load_balancers"] != "" {
		autoScalingGroupOpts.LoadBalancerNames = expandStringList(flatmap.Expand(
			rs.Attributes, "load_balancers").([]interface{}))
	}

	if rs.Attributes["vpc_identifier"] != "" {
		autoScalingGroupOpts.VPCZoneIdentifier = expandStringList(flatmap.Expand(
			rs.Attributes, "vpc_identifier").([]interface{}))
	}

	autoScalingGroupOpts.Name = rs.Attributes["name"]
	autoScalingGroupOpts.HealthCheckType = rs.Attributes["health_check_type"]
	autoScalingGroupOpts.LaunchConfigurationName = rs.Attributes["launch_configuration"]

	log.Printf("[DEBUG] autoscaling Group create configuration: %#v", autoScalingGroupOpts)
	_, err = autoscalingconn.CreateAutoScalingGroup(&autoScalingGroupOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating autoscaling Group: %s", err)
	}

	rs.ID = rs.Attributes["name"]

	log.Printf("[INFO] autoscaling Group ID: %s", rs.ID)

	g, err := resource_aws_autoscaling_group_retrieve(rs.ID, autoscalingconn)
	if err != nil {
		return rs, err
	}

	return resource_aws_autoscaling_group_update_state(rs, g)
}

func resource_aws_autoscaling_group_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	rs := s.MergeDiff(d)
	log.Printf("ResourceDiff: %s", d)
	log.Printf("ResourceState: %s", s)
	log.Printf("Merged: %s", rs)

	return nil, fmt.Errorf("Did not update")
}

func resource_aws_autoscaling_group_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	// p := meta.(*ResourceProvider)
	// autoscalingconn := p.autoscalingconn

	log.Printf("[DEBUG] autoscaling Group destroy: %v", s.ID)

	// err := nil

	// _, err := autoscalingconn.DeleteAutoScalingGroup(autoscaling.autoscalingGroup{Id: s.ID})

	// if err != nil {
	// 	autoscalingerr, ok := err.(*autoscaling.Error)
	// 	if ok && autoscalingerr.Code == "InvalidGroup.NotFound" {
	// 		return nil
	// 	}
	// }

	return nil
}

func resource_aws_autoscaling_group_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	g, err := resource_aws_autoscaling_group_retrieve(s.ID, autoscalingconn)

	if err != nil {
		return s, err
	}

	return resource_aws_autoscaling_group_update_state(s, g)
}

func resource_aws_autoscaling_group_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"min_size":                  diff.AttrTypeCreate,
			"max_size":                  diff.AttrTypeCreate,
			"default_cooldown":          diff.AttrTypeCreate,
			"name":                      diff.AttrTypeCreate,
			"desired_capicity":          diff.AttrTypeCreate,
			"health_check_grace_period": diff.AttrTypeCreate,
			"health_check_type":         diff.AttrTypeCreate,
			"launch_configuration":      diff.AttrTypeCreate,
			"vpc_zone_identifier":       diff.AttrTypeCreate,
			"load_balancers":            diff.AttrTypeCreate,
			"availability_zones":        diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{},
	}

	return b.Diff(s, c)
}

func resource_aws_autoscaling_group_update_state(
	s *terraform.ResourceState,
	g *autoscaling.AutoScalingGroup) (*terraform.ResourceState, error) {

	s.Attributes["min_size"] = strconv.Itoa(g.MinSize)
	s.Attributes["max_size"] = strconv.Itoa(g.MaxSize)
	s.Attributes["default_cooldown"] = strconv.Itoa(g.DefaultCooldown)
	s.Attributes["name"] = g.Name
	s.Attributes["desired_capacity"] = strconv.Itoa(g.DesiredCapacity)
	s.Attributes["health_check_grace_period"] = strconv.Itoa(g.HealthCheckGracePeriod)
	s.Attributes["health_check_type"] = g.HealthCheckType
	s.Attributes["launch_configuration"] = g.LaunchConfigurationName
	s.Attributes["vpc_zone_identifier"] = g.VPCZoneIdentifier

	// Flatten our sg values
	toFlatten := make(map[string]interface{})
	toFlatten["load_balancers"] = flattenLoadBalancers(g.LoadBalancerNames)
	toFlatten["availability_zones"] = flattenAvailabilityZones(g.AvailabilityZones)

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

// Returns a single group by it's ID
func resource_aws_autoscaling_group_retrieve(id string, autoscalingconn *autoscaling.AutoScaling) (*autoscaling.AutoScalingGroup, error) {
	describeOpts := autoscaling.DescribeAutoScalingGroups{
		Names: []string{id},
	}

	log.Printf("[DEBUG] autoscaling Group describe configuration: %#v", describeOpts)

	describeGroups, err := autoscalingconn.DescribeAutoScalingGroups(&describeOpts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving autoscaling groups: %s", err)
	}

	// Verify AWS returned our sg
	if len(describeGroups.AutoScalingGroups) != 1 ||
		describeGroups.AutoScalingGroups[0].Name != id {
		if err != nil {
			return nil, fmt.Errorf("Unable to find autoscaling group: %#v", describeGroups.AutoScalingGroups)
		}
	}

	g := describeGroups.AutoScalingGroups[0]

	return &g, nil
}