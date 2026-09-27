/*
 * mdbl.c - Minimal C bootstrap for Moddable XS machine initialization in RePebble Alloy
 */

#include <pebble.h>

void xs_init(void) {
    // Moddable XS Virtual Machine runtime bootstrap entrypoint
}

int main(void) {
    xs_init();
    app_event_loop();
    return 0;
}
