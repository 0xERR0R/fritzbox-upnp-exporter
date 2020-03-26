package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_filterConvertAndCorrectByService(t *testing.T) {
	sut := newFritzBoxCollector(&Config{})

	val := sut.filterConvertAndCorrectByService([]serviceActionValue{
		serviceActionValue{actionName: "action",
			serviceType: "service",
			value:       "4711",
			variable:    "var"},
	}, "service", "action", "var")

	assert.Equal(t, float64(4711), val)

	val = sut.filterConvertAndCorrectByService([]serviceActionValue{
		serviceActionValue{actionName: "action",
			serviceType: "service",
			value:       "4712",
			variable:    "var"},
	}, "service", "action", "var")

	assert.Equal(t, float64(4712), val)

	val = sut.filterConvertAndCorrectByService([]serviceActionValue{
		serviceActionValue{actionName: "action",
			serviceType: "service",
			value:       "100",
			variable:    "var"},
	}, "service", "action", "var")

	assert.Equal(t, float64(0), val)

	val = sut.filterConvertAndCorrectByService([]serviceActionValue{
		serviceActionValue{actionName: "action",
			serviceType: "service",
			value:       "150",
			variable:    "var"},
	}, "service", "action", "var")

	assert.Equal(t, float64(50), val)

	val = sut.filterConvertAndCorrectByService([]serviceActionValue{
		serviceActionValue{actionName: "action",
			serviceType: "service",
			value:       "120",
			variable:    "var"},
	}, "service", "action", "var")

	assert.Equal(t, float64(0), val)

	val = sut.filterConvertAndCorrectByService([]serviceActionValue{
		serviceActionValue{actionName: "action",
			serviceType: "service",
			value:       "150",
			variable:    "var"},
	}, "service", "action", "var")

	assert.Equal(t, float64(30), val)

}
