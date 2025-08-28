package digitalocean

import "fmt"
import "strconv"

type DigitalOceanTaskConfig map[string]string

func (d DigitalOceanTaskConfig) ValidateBasic() error {
	if v, ok := d["region"]; !ok || v == "" {
		return fmt.Errorf("region has to be non-empty and a string")
	}

	if v, ok := d["size"]; !ok || v == "" {
		return fmt.Errorf("size has to be non-empty and a string")
	}

	if v, ok := d["image_id"]; !ok || v == "" {
		return fmt.Errorf("image_id has to be non-empty")
	}

	if _, err := strconv.ParseInt(d["image_id"], 10, 64); err != nil {
		return fmt.Errorf("image_id has to be an integer")
	}

	return nil
}
