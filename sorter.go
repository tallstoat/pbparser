package pbparser

import "sort"

func sortMessage(msg MessageElement) {
	sort.Slice(msg.OneOfs, func(i, j int) bool {
		return msg.OneOfs[i].Name < msg.OneOfs[j].Name
	})

	sort.Slice(msg.Enums, func(i, j int) bool {
		return msg.Enums[i].Name < msg.Enums[j].Name
	})

	for _, child := range msg.Messages {
		sortMessage(child)
	}
}

// Sort sorts every protofile component alphabetically.
// Not fully implemented (options and extensions most notably).
func (pf *ProtoFile) Sort() {
	sort.Strings(pf.PublicDependencies)
	sort.Strings(pf.Dependencies)

	sort.Slice(pf.Services, func(i, j int) bool {
		return pf.Services[i].Name < pf.Services[j].Name
	})

	sort.Slice(pf.Enums, func(i, j int) bool {
		return pf.Enums[i].Name < pf.Enums[j].Name
	})

	sort.Slice(pf.Messages, func(i, j int) bool {
		return pf.Messages[i].Name < pf.Messages[j].Name
	})
	for _, message := range pf.Messages {
		sortMessage(message)
	}
}
