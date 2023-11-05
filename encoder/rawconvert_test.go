package encoder

import (
	"testing"
)

func TestConvertRawToJPG(t *testing.T) {
	testCases := []struct {
		rawPath        string
		optimizedPath  string
		expectedResult string
		expectedStatus bool
	}{
		// blackbird.NEF is from https://github.com/jewright/nef-to-jpg/blob/main/photoconverter/Sample-Images/blackbird.NEF
		{"../pics/blackbird.NEF", "../exhaust_test/", "../exhaust_test/blackbird.NEF_extracted.jpg", true},
		{"../pics/big.jpg", "../exhaust_test/", "../pics/big.jpg", false},
	}

	for _, tc := range testCases {
		result, status := ConvertRawToJPG(tc.rawPath, tc.optimizedPath)

		if result != tc.expectedResult || status != tc.expectedStatus {
			t.Errorf("ConvertRawToJPG(%s, %s) => (%s, %t), expected (%s, %t)", tc.rawPath, tc.optimizedPath, result, status, tc.expectedResult, tc.expectedStatus)
		}
	}
}
