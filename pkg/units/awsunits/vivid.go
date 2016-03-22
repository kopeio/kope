package awsunits

type DistroVivid struct {
}

func (d *DistroVivid) GetImageID(region string) string {
	// This is the ubuntu 15.04 image for <region>, amd64, hvm:ebs-ssd
	// See here: http://cloud-images.ubuntu.com/locator/ec2/ for other images
	// This will need to be updated from time to time as amis are deprecated
	switch region {
	case "ap-northeast-1":
		return "ami-907fa690"
	case "ap-southeast-1":
		return "ami-b4a79de6"

	case "eu-central-1":
		return "ami-e8635bf5"
	case "eu-west-1":
		return "ami-0fd0ae78"
	case "sa-east-1":
		return "ami-f9f675e4"
	case "us-east-1":
		return "ami-f57b8f9e"
	case "us-west-1":

		return "ami-87b643c3"
	case "cn-north-1":
		return "ami-3abf2203"
	case "ap-southeast-2":
		return "ami-1bb9c221"
	case "us-west-2":
		return "ami-33566d03"
	default:
		return ""
	}
}
