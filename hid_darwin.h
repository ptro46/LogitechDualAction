#ifndef DUAL_ACTION_H
#define DUAL_ACTION_H
#include <stdint.h>
typedef struct { int page, usage, valid; int64_t value, min, max; } Element;
typedef struct { char name[256]; int vendor, product, count; Element elements[256]; } Snapshot;
int hid_init(int vendor, int product);
int hid_snapshot(Snapshot *out);
void hid_list(void);
void hid_close(void);
#endif
