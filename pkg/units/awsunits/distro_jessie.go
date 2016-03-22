package awsunits

import (
	"github.com/kopeio/kope/pkg/fi"
	"github.com/aws/aws-sdk-go/service/ec2"
	"fmt"
)

type DistroJessie struct {
}

func (d *DistroJessie) GetImageID(c *fi.Context) (string, error) {
	// TODO: publish on a k8s AWS account
	owner := "721322707521"
	// TODO: we could use tags for the image
	awsImageName := "k8s-1.2-debian-jessie-amd64-hvm-2016-03-05-ebs"

	cloud := c.Cloud().(*fi.AWSCloud)

	var filters []*ec2.Filter
	filters = append(filters, fi.NewEC2Filter("name", awsImageName))
	request := &ec2.DescribeImagesInput{
		Owners: []*string{&owner},
		Filters: filters,
	}

	response, err := cloud.EC2.DescribeImages(request)
	if err != nil {
		return "", fmt.Errorf("error listing EC2 images: %v", err)
	}

	for _, image := range response.Images {
		return *image.ImageId, nil
	}

	return "", fmt.Errorf("cannot determine the AWS image to use")
}
