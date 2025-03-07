// Code generated by go-swagger; DO NOT EDIT.

package headchef_operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	headchef_models "github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

// ArtifactJobStatusReader is a Reader for the ArtifactJobStatus structure.
type ArtifactJobStatusReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ArtifactJobStatusReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewArtifactJobStatusOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 404:
		result := NewArtifactJobStatusNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		result := NewArtifactJobStatusDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewArtifactJobStatusOK creates a ArtifactJobStatusOK with default headers values
func NewArtifactJobStatusOK() *ArtifactJobStatusOK {
	return &ArtifactJobStatusOK{}
}

/*ArtifactJobStatusOK handles this case with default header values.

Job completion has been recorded
*/
type ArtifactJobStatusOK struct {
}

func (o *ArtifactJobStatusOK) Error() string {
	return fmt.Sprintf("[POST /artifacts/{artifact_id}/job-status][%d] artifactJobStatusOK ", 200)
}

func (o *ArtifactJobStatusOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewArtifactJobStatusNotFound creates a ArtifactJobStatusNotFound with default headers values
func NewArtifactJobStatusNotFound() *ArtifactJobStatusNotFound {
	return &ArtifactJobStatusNotFound{}
}

/*ArtifactJobStatusNotFound handles this case with default header values.

No artifact exists with the request artifact ID
*/
type ArtifactJobStatusNotFound struct {
	Payload *headchef_models.RestAPIError
}

func (o *ArtifactJobStatusNotFound) Error() string {
	return fmt.Sprintf("[POST /artifacts/{artifact_id}/job-status][%d] artifactJobStatusNotFound  %+v", 404, o.Payload)
}

func (o *ArtifactJobStatusNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(headchef_models.RestAPIError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewArtifactJobStatusDefault creates a ArtifactJobStatusDefault with default headers values
func NewArtifactJobStatusDefault(code int) *ArtifactJobStatusDefault {
	return &ArtifactJobStatusDefault{
		_statusCode: code,
	}
}

/*ArtifactJobStatusDefault handles this case with default header values.

If there is an error processing the request
*/
type ArtifactJobStatusDefault struct {
	_statusCode int

	Payload *headchef_models.RestAPIError
}

// Code gets the status code for the artifact job status default response
func (o *ArtifactJobStatusDefault) Code() int {
	return o._statusCode
}

func (o *ArtifactJobStatusDefault) Error() string {
	return fmt.Sprintf("[POST /artifacts/{artifact_id}/job-status][%d] artifactJobStatus default  %+v", o._statusCode, o.Payload)
}

func (o *ArtifactJobStatusDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(headchef_models.RestAPIError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
