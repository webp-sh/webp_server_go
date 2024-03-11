package encoder

import (
	"testing"
	"webp_server_go/config"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestResizeImage(t *testing.T) {
	img, _ := vips.Black(500, 500)

	// Define the parameters for the test cases
	testCases := []struct {
		extraParams config.ExtraParams // Extra parameters
		expectedH   int                // Expected height
		expectedW   int                // Expected width
	}{
		// Tests for MaxHeight and MaxWidth
		// Both extraParams.MaxHeight and extraParams.MaxWidth are 0
		{
			extraParams: config.ExtraParams{
				MaxHeight: 0,
				MaxWidth:  0,
			},
			expectedH: 500,
			expectedW: 500,
		},
		// Both extraParams.MaxHeight and extraParams.MaxWidth are greater than 0, but the image size is smaller than the limits
		{
			extraParams: config.ExtraParams{
				MaxHeight: 1000,
				MaxWidth:  1000,
			},
			expectedH: 500,
			expectedW: 500,
		},
		// Both extraParams.MaxHeight and extraParams.MaxWidth are greater than 0, and the image exceeds the limits
		{
			extraParams: config.ExtraParams{
				MaxHeight: 200,
				MaxWidth:  200,
			},
			expectedH: 200,
			expectedW: 200,
		},
		// Only MaxHeight is set to 200
		{
			extraParams: config.ExtraParams{
				MaxHeight: 200,
				MaxWidth:  0,
			},
			expectedH: 200,
			expectedW: 200,
		},

		// Test for Width and Height
		{
			extraParams: config.ExtraParams{
				Width:  200,
				Height: 200,
			},
			expectedH: 200,
			expectedW: 200,
		},
		{
			extraParams: config.ExtraParams{
				Width:  200,
				Height: 500,
			},
			expectedH: 500,
			expectedW: 200,
		},
	}

	// Iterate through the test cases and perform the tests
	for _, tc := range testCases {
		err := resizeImage(img, tc.extraParams)
		if err != nil {
			t.Errorf("resizeImage failed with error: %v", err)
		}

		// Verify if the adjusted image height and width match the expected values
		actualH := img.Height()
		actualW := img.Width()
		if actualH != tc.expectedH || actualW != tc.expectedW {
			t.Errorf("resizeImage failed: expected (%d, %d), got (%d, %d)", tc.expectedH, tc.expectedW, actualH, actualW)
		}
	}
}
