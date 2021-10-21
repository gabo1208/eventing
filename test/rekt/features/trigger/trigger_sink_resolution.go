/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trigger

import (
	"fmt"

	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/eventing/test/rekt/resources/delivery"
	"knative.dev/eventing/test/rekt/resources/eventlibrary"
	"knative.dev/eventing/test/rekt/resources/trigger"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/feature"
)

// SourceToTriggerSinkWithDLS tests to see if a Ready Trigger with a DLS defined send
// failing events to it's DLS.
//
// source ---> broker --[trigger]--> bad uri
//                          |
//                          +--[DLS]--> sink
//
func SourceToTriggerSinkWithDLS(triggerName string) *feature.Feature {
	prober := eventshub.NewProber()
	brokerName := feature.MakeRandomK8sName("broker-")
	prober.SetTargetResource(broker.GVR(), brokerName)

	f := feature.NewFeature()

	lib := feature.MakeRandomK8sName("lib")
	f.Setup("install events", eventlibrary.Install(lib))
	f.Setup("event cache is ready", eventlibrary.IsReady(lib))
	f.Setup("use events cache", prober.SenderEventsFromSVC(lib, "events/three.ce"))
	if err := prober.ExpectYAMLEvents(eventlibrary.PathFor("events/three.ce")); err != nil {
		panic(fmt.Errorf("can not find event files: %s", err))
	}

	// Setup Probes
	f.Setup("install recorder", prober.ReceiverInstall("sink"))

	// Setup data plane
	f.Setup("update broker with DLS", broker.Install(
		brokerName,
		broker.WithEnvConfig()...,
	))

	f.Setup("install trigger", trigger.Install(
		triggerName,
		brokerName,
		trigger.WithSubscriber(nil, "bad://uri"),
		delivery.WithDeadLetterSink(prober.AsKReference("sink"), "")))

	// Resources ready.
	f.Setup("trigger goes ready", trigger.IsReady(triggerName))

	// Install events after data plane is ready.
	f.Setup("install source", prober.SenderInstall("source"))

	// After we have finished sending.
	f.Requirement("sender is finished", prober.SenderDone("source"))

	// Assert events ended up where we expected.
	f.Stable("trigger with DLS").
		Must("accepted all events", prober.AssertSentAll("source")).
		Must("deliver event to DLS", prober.AssertReceivedAll("source", "sink"))

	return f
}

// SourceToTriggerSinkWithDLSDontUseBrokers tests to see if a Ready Trigger sends
// failing events to it's DLS even when it's corresponding Ready Broker also have a DLS defined.
//
// source ---> broker --[trigger]--> bad uri
//               |				  |
//               +--[DLS]   +--[DLS]--> sink
//
func SourceToTriggerSinkWithDLSDontUseBrokers(triggerName string) *feature.Feature {
	prober := eventshub.NewProber()
	brokerName := feature.MakeRandomK8sName("broker-")
	prober.SetTargetResource(broker.GVR(), brokerName)

	f := feature.NewFeature()

	lib := feature.MakeRandomK8sName("lib")
	f.Setup("install events", eventlibrary.Install(lib))
	f.Setup("event cache is ready", eventlibrary.IsReady(lib))
	f.Setup("use events cache", prober.SenderEventsFromSVC(lib, "events/three.ce"))
	if err := prober.ExpectYAMLEvents(eventlibrary.PathFor("events/three.ce")); err != nil {
		panic(fmt.Errorf("can not find event files: %s", err))
	}

	// Setup Probes
	f.Setup("install trigger recorder", prober.ReceiverInstall("trigger-sink"))
	f.Setup("install brokers recorder", prober.ReceiverInstall("broker-sink"))

	// Setup data plane
	brokerConfig := append(
		broker.WithEnvConfig(),
		delivery.WithDeadLetterSink(prober.AsKReference("broker-sink"), ""))
	f.Setup("update broker with DLS", broker.Install(
		brokerName,
		brokerConfig...,
	))

	f.Setup("install trigger", trigger.Install(
		triggerName,
		brokerName,
		trigger.WithSubscriber(nil, "bad://uri"),
		delivery.WithDeadLetterSink(prober.AsKReference("trigger-sink"), "")))

	// Resources ready.
	f.Setup("trigger goes ready", trigger.IsReady(triggerName))

	// Install events after data plane is ready.
	f.Setup("install source", prober.SenderInstall("source"))

	// After we have finished sending.
	f.Requirement("sender is finished", prober.SenderDone("source"))

	// Assert events ended up where we expected.
	f.Stable("trigger with DLS").
		Must("accepted all events", prober.AssertSentAll("source")).
		Must("deliver events to trigger DLS", prober.AssertReceivedAll("source", "trigger-sink"))

	return f
}
