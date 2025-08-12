package common_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/common"
)

var _ = Describe("UnixTime", func() {
	Describe("UnmarshalJSON", func() {
		Context("when unmarshaling numeric Unix timestamp", func() {
			It("should handle integer timestamp", func() {
				data := []byte("1755005925")
				var ut common.UnixTime
				err := json.Unmarshal(data, &ut)
				Expect(err).ToNot(HaveOccurred())
				
				// Check the time is correct (2025-08-11 13:18:45 UTC)
				expectedTime := time.Unix(1755005925, 0)
				Expect(ut.Time.Unix()).To(Equal(expectedTime.Unix()))
			})

			It("should handle timestamp with fractional seconds", func() {
				data := []byte("1755005925.233")
				var ut common.UnixTime
				err := json.Unmarshal(data, &ut)
				Expect(err).ToNot(HaveOccurred())
				
				// Check the time including nanoseconds
				expectedTime := time.Unix(1755005925, 233000000)
				Expect(ut.Time.UnixNano()).To(Equal(expectedTime.UnixNano()))
			})

			It("should handle LocalStack response format", func() {
				// Actual LocalStack response
				response := `{"LastModifiedDate": 1755005925.233, "Name": "/myapp/prod/database-url"}`
				
				type TestStruct struct {
					LastModifiedDate *common.UnixTime `json:"LastModifiedDate,omitempty"`
					Name             *string          `json:"Name,omitempty"`
				}
				
				var result TestStruct
				err := json.Unmarshal([]byte(response), &result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.LastModifiedDate).ToNot(BeNil())
				Expect(result.LastModifiedDate.Unix()).To(Equal(int64(1755005925)))
			})
		})

		Context("when unmarshaling RFC3339 string", func() {
			It("should handle RFC3339 format", func() {
				data := []byte(`"2025-08-12T13:18:45Z"`)
				var ut common.UnixTime
				err := json.Unmarshal(data, &ut)
				Expect(err).ToNot(HaveOccurred())
				
				// Parse the expected time to verify
				expectedTime, _ := time.Parse(time.RFC3339, "2025-08-12T13:18:45Z")
				Expect(ut.Time.Unix()).To(Equal(expectedTime.Unix()))
			})
		})

		Context("when unmarshaling invalid data", func() {
			It("should return error for invalid format", func() {
				data := []byte(`{"invalid": "object"}`)
				var ut common.UnixTime
				err := json.Unmarshal(data, &ut)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
			})
		})
	})

	Describe("MarshalJSON", func() {
		Context("when marshaling UnixTime", func() {
			It("should marshal as numeric Unix timestamp", func() {
				ut := common.UnixTime{Time: time.Unix(1755005925, 233000000)}
				data, err := json.Marshal(ut)
				Expect(err).ToNot(HaveOccurred())
				
				// Should be numeric format with 3 decimal places
				Expect(string(data)).To(Equal("1755005925.233"))
			})

			It("should handle zero time", func() {
				ut := common.UnixTime{}
				data, err := json.Marshal(ut)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("null"))
			})
		})
	})

	Describe("Helper functions", func() {
		It("should convert between UnixTime and time.Time", func() {
			now := time.Now()
			ut := common.NewUnixTime(now)
			Expect(ut.Time).To(Equal(now))
			
			timePtr := ut.ToTime()
			Expect(*timePtr).To(Equal(now))
			
			utFromTime := common.FromTime(&now)
			Expect(utFromTime.Time).To(Equal(now))
		})

		It("should handle nil pointers", func() {
			var ut *common.UnixTime
			Expect(ut.ToTime()).To(BeNil())
			
			var t *time.Time
			Expect(common.FromTime(t)).To(BeNil())
		})
	})
})