package main

const (
	// Created indicates the job has just been created
	Created = iota

	// ReceivingInputs means some inputs (>0) have been received
	ReceivingInputs

	// AllInputReceived means no further inputs will be received
	AllInputReceived

	// AllOutputReceived means all outputs have been received
	AllOutputReceived

	// ErrorState indicates this job is in error, no more interaction should occur
	ErrorState
)
