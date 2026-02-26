package v1alpha1

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	v1 "github.com/openclawrocks/k8s-operator/api/v1"
)

// ConvertTo converts this OpenClawInstance (v1alpha1) to the hub version (v1).
func (o *OpenClawInstance) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1.OpenClawInstance)
	if !ok {
		return fmt.Errorf("expected *v1.OpenClawInstance, got %T", dstRaw)
	}

	// JSON round-trip: specs are identical between versions.
	data, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("marshaling v1alpha1 OpenClawInstance: %w", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("unmarshaling to v1 OpenClawInstance: %w", err)
	}

	return nil
}

// ConvertFrom converts from the hub version (v1) to this OpenClawInstance (v1alpha1).
func (o *OpenClawInstance) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1.OpenClawInstance)
	if !ok {
		return fmt.Errorf("expected *v1.OpenClawInstance, got %T", srcRaw)
	}

	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshaling v1 OpenClawInstance: %w", err)
	}
	if err := json.Unmarshal(data, o); err != nil {
		return fmt.Errorf("unmarshaling to v1alpha1 OpenClawInstance: %w", err)
	}

	return nil
}

// ConvertTo converts this OpenClawSelfConfig (v1alpha1) to the hub version (v1).
func (o *OpenClawSelfConfig) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1.OpenClawSelfConfig)
	if !ok {
		return fmt.Errorf("expected *v1.OpenClawSelfConfig, got %T", dstRaw)
	}

	data, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("marshaling v1alpha1 OpenClawSelfConfig: %w", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("unmarshaling to v1 OpenClawSelfConfig: %w", err)
	}

	return nil
}

// ConvertFrom converts from the hub version (v1) to this OpenClawSelfConfig (v1alpha1).
func (o *OpenClawSelfConfig) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1.OpenClawSelfConfig)
	if !ok {
		return fmt.Errorf("expected *v1.OpenClawSelfConfig, got %T", srcRaw)
	}

	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshaling v1 OpenClawSelfConfig: %w", err)
	}
	if err := json.Unmarshal(data, o); err != nil {
		return fmt.Errorf("unmarshaling to v1alpha1 OpenClawSelfConfig: %w", err)
	}

	return nil
}
