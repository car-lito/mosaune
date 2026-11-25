package main

import musicresults "mosaune/extensionkit/capabilities/musicresults/v1"

func (*musicBrainzProvider) ImageReferenceKinds(ref string) []string {
	if (musicresults.VisualQuery{Operation: "artwork", Ref: ref}).Validate() == nil {
		return []string{"music"}
	}
	return nil
}
