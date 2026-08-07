//go:build darwin && cgo

package fonts

/*
#cgo LDFLAGS: -framework CoreText -framework CoreFoundation
#include <CoreText/CoreText.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

static int appendFontName(char **output, size_t *length, size_t *capacity, const char *name) {
	size_t nameLength = strlen(name);
	if (*length + nameLength + 2 > *capacity) {
		size_t nextCapacity = *capacity * 2;
		while (*length + nameLength + 2 > nextCapacity) {
			nextCapacity *= 2;
		}
		char *next = realloc(*output, nextCapacity);
		if (next == NULL) {
			return 0;
		}
		*output = next;
		*capacity = nextCapacity;
	}
	memcpy(*output + *length, name, nameLength);
	*length += nameLength;
	(*output)[(*length)++] = '\n';
	(*output)[*length] = '\0';
	return 1;
}

static char *taskaiMonospaceFontFamilies(void) {
	CFArrayRef families = CTFontManagerCopyAvailableFontFamilyNames();
	if (families == NULL) {
		return NULL;
	}
	size_t capacity = 256;
	size_t length = 0;
	char *output = malloc(capacity);
	if (output == NULL) {
		CFRelease(families);
		return NULL;
	}
	output[0] = '\0';
	CFIndex count = CFArrayGetCount(families);
	for (CFIndex index = 0; index < count; index++) {
		CFStringRef family = (CFStringRef)CFArrayGetValueAtIndex(families, index);
		CTFontRef font = CTFontCreateWithName(family, 12.0, NULL);
		if (font == NULL) {
			continue;
		}
		CTFontSymbolicTraits traits = CTFontGetSymbolicTraits(font);
		if ((traits & kCTFontMonoSpaceTrait) != 0) {
			CFIndex maximum = CFStringGetMaximumSizeForEncoding(CFStringGetLength(family), kCFStringEncodingUTF8) + 1;
			char *name = malloc((size_t)maximum);
			if (name != NULL) {
				if (CFStringGetCString(family, name, maximum, kCFStringEncodingUTF8)) {
					appendFontName(&output, &length, &capacity, name);
				}
				free(name);
			}
		}
		CFRelease(font);
	}
	CFRelease(families);
	return output;
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

func systemCandidates() []Candidate {
	raw := C.taskaiMonospaceFontFamilies()
	if raw == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(raw))

	candidates := make([]Candidate, 0)
	for _, family := range strings.Split(C.GoString(raw), "\n") {
		candidates = append(candidates, Candidate{Family: family, Spacing: SpacingMono})
	}
	return candidates
}
