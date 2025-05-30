package models

type UsageType string
type DiscountType string
type DiscountTarget string

const (
	UsageTypeSingleUse UsageType = "single_use"
	UsageTypeMultiUse  UsageType = "multi_use"

	DiscountTypeFlat      DiscountType = "flat"
	DiscountTypePercentage DiscountType = "percentage"

	DiscountTargetDelivery DiscountTarget = "delivery"
	DiscountTargetOrder    DiscountTarget = "total_order_value"
)
