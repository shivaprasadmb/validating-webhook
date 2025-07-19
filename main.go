package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// admissionReviewHandler is the main handler for our webhook. It decodes the
// incoming request, validates the pod, and sends back a response.
func admissionReviewHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Read and decode the AdmissionReview request from the API server.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read request body", http.StatusBadRequest)
		log.Printf("Error reading body: %v", err)
		return
	}
	defer r.Body.Close()

	// We create an AdmissionReview object to hold the request.
	admissionReviewReq := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReviewReq); err != nil {
		http.Error(w, "could not unmarshal request", http.StatusBadRequest)
		log.Printf("Error unmarshaling request: %v", err)
		return
	}
	
	log.Println("Received admission review request")

	// 2. Prepare the AdmissionReview response. It's crucial to copy the UID
	// from the request to the response, so the API server knows which request
	// this response is for.
	admissionReviewResp := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionv1.AdmissionResponse{
			UID: admissionReviewReq.Request.UID,
		},
	}
	
	// By default, we allow the request.
	admissionReviewResp.Response.Allowed = true

	// 3. Perform the actual validation.
	// We only care about Pods. The Kind is part of the AdmissionRequest.
	if admissionReviewReq.Request.Kind.Kind == "Pod" {
		pod := corev1.Pod{}
		// The actual object is sent in Raw format, so we need to unmarshal it.
		if err := json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod); err != nil {
			http.Error(w, "could not unmarshal pod object", http.StatusBadRequest)
			log.Printf("Error unmarshaling pod: %v", err)
			return
		}

		// Our validation logic: check for the 'app' label.
		log.Printf("Validating pod: %s/%s", pod.Namespace, pod.Name)
		if _, ok := pod.Labels["app"]; !ok {
			// If validation fails, we deny the request and provide a reason.
			admissionReviewResp.Response.Allowed = false
			admissionReviewResp.Response.Result = &metav1.Status{
				Message: "Pod rejected: missing 'app' label.",
				Code:    http.StatusForbidden, // A suitable HTTP status code.
				Reason:  metav1.StatusReasonForbidden,
			}
			log.Printf("Pod %s/%s rejected: missing 'app' label", pod.Namespace, pod.Name)
		} else {
			log.Printf("Pod %s/%s allowed", pod.Namespace, pod.Name)
		}
	}


	// 4. Encode the AdmissionReview response and send it back to the API server.
	respBytes, err := json.Marshal(admissionReviewResp)
	if err != nil {
		http.Error(w, "could not marshal response", http.StatusInternalServerError)
		log.Printf("Error marshaling response: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(respBytes); err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func main() {
	// Paths to the TLS certificate and key files.
	// Kubernetes API server requires the webhook to be served over HTTPS.
	certFile := "/etc/webhook/certs/tls.crt"
	keyFile := "/etc/webhook/certs/tls.key"

	http.HandleFunc("/validate", admissionReviewHandler)
	
	log.Println("Starting webhook server on port 8443...")

	// ListenAndServeTLS starts an HTTPS server.
	err := http.ListenAndServeTLS(":8443", certFile, keyFile, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}