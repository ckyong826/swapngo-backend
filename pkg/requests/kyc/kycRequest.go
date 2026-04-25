package kyc

type SubmitKYCRequest struct {
	FullName     string `json:"full_name" binding:"required"`
	ICNumber     string `json:"ic_number" binding:"required"`
	ICFrontPhoto string `json:"ic_front_photo" binding:"required"` // base64-encoded image
	ICBackPhoto  string `json:"ic_back_photo" binding:"required"`  // base64-encoded image
}
