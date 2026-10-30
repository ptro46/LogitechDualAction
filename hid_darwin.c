//go:build darwin && cgo

#include "hid_darwin.h"
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/hid/IOHIDLib.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static IOHIDManagerRef manager;
static int number(IOHIDDeviceRef d, CFStringRef key) {
    int n = 0;
    CFTypeRef v = IOHIDDeviceGetProperty(d, key);
    if (v && CFGetTypeID(v) == CFNumberGetTypeID()) CFNumberGetValue(v, kCFNumberIntType, &n);
    return n;
}
static void name(IOHIDDeviceRef d, char *out, size_t size) {
    CFTypeRef v = IOHIDDeviceGetProperty(d, CFSTR(kIOHIDProductKey));
    snprintf(out, size, "Périphérique HID");
    if (v && CFGetTypeID(v) == CFStringGetTypeID()) CFStringGetCString(v, out, size, kCFStringEncodingUTF8);
}
int hid_init(int vendor, int product) {
    manager = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
    if (!manager) return -1;
    CFMutableDictionaryRef match = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    if (vendor >= 0) {
        CFNumberRef v = CFNumberCreate(NULL, kCFNumberIntType, &vendor);
        CFNumberRef p = CFNumberCreate(NULL, kCFNumberIntType, &product);
        CFDictionarySetValue(match, CFSTR(kIOHIDVendorIDKey), v);
        CFDictionarySetValue(match, CFSTR(kIOHIDProductIDKey), p);
        CFRelease(v); CFRelease(p);
    }
    IOHIDManagerSetDeviceMatching(manager, match);
    CFRelease(match);
    IOHIDManagerScheduleWithRunLoop(manager, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
    return (int)IOHIDManagerOpen(manager, kIOHIDOptionsTypeNone);
}
static void pump(void) {
    CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.001, false);
}
int hid_snapshot(Snapshot *out) {
    memset(out, 0, sizeof(*out));
    pump();
    CFSetRef devices = IOHIDManagerCopyDevices(manager);
    if (!devices) return 0;
    CFIndex count = CFSetGetCount(devices);
    if (!count) { CFRelease(devices); return 0; }
    IOHIDDeviceRef *items = calloc((size_t)count, sizeof(*items));
    if (!items) { CFRelease(devices); return -1; }
    CFSetGetValues(devices, (const void **)items);
    IOHIDDeviceRef device = items[0];
    name(device, out->name, sizeof(out->name));
    out->vendor = number(device, CFSTR(kIOHIDVendorIDKey));
    out->product = number(device, CFSTR(kIOHIDProductIDKey));
    CFArrayRef elements = IOHIDDeviceCopyMatchingElements(device, NULL, kIOHIDOptionsTypeNone);
    if (elements) {
        for (CFIndex i = 0; i < CFArrayGetCount(elements); i++) {
            IOHIDElementRef element = (IOHIDElementRef)CFArrayGetValueAtIndex(elements, i);
            IOHIDElementType type = IOHIDElementGetType(element);
            if (type < kIOHIDElementTypeInput_Misc || type > kIOHIDElementTypeInput_ScanCodes) continue;
            int page = IOHIDElementGetUsagePage(element), usage = IOHIDElementGetUsage(element);
            if (page != 9 && !(page == 1 && usage >= 0x30 && usage <= 0x39)) continue;
            if (out->count >= 256) break;
            Element *e = &out->elements[out->count++];
            e->page = page; e->usage = usage;
            e->min = IOHIDElementGetLogicalMin(element); e->max = IOHIDElementGetLogicalMax(element);
            IOHIDValueRef value = NULL;
            e->valid = IOHIDDeviceGetValue(device, element, &value) == kIOReturnSuccess && value;
            if (e->valid) e->value = IOHIDValueGetIntegerValue(value);
        }
        CFRelease(elements);
    }
    free(items); CFRelease(devices);
    return out->count ? 1 : -1;
}
void hid_list(void) {
    pump();
    CFSetRef devices = IOHIDManagerCopyDevices(manager);
    int found = 0;
    if (devices) {
        CFIndex count = CFSetGetCount(devices);
        IOHIDDeviceRef *items = calloc((size_t)(count ? count : 1), sizeof(*items));
        if (!items) { CFRelease(devices); return; }
        CFSetGetValues(devices, (const void **)items);
        for (CFIndex i = 0; i < count; i++) {
            IOHIDDeviceRef d = items[i];
            if (!IOHIDDeviceConformsTo(d, 1, 4) && !IOHIDDeviceConformsTo(d, 1, 5) && !IOHIDDeviceConformsTo(d, 1, 8)) continue;
            char label[256]; name(d, label, sizeof(label));
            printf("%04x:%04x  %s\n", number(d, CFSTR(kIOHIDVendorIDKey)), number(d, CFSTR(kIOHIDProductIDKey)), label);
            found++;
        }
        free(items); CFRelease(devices);
    }
    if (!found) puts("Aucune manette HID détectée.");
    fflush(stdout);
}
void hid_close(void) {
    if (!manager) return;
    IOHIDManagerUnscheduleFromRunLoop(manager, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
    IOHIDManagerClose(manager, kIOHIDOptionsTypeNone);
    CFRelease(manager); manager = NULL;
}
