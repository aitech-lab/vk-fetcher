
// ॐ //

#pragma once

// An dynamic null terminated buffer

#include <stdlib.h>

typedef struct {
    char*  buf;
    size_t size;
} buf_t;

buf_t* buf_new();
void buf_init(buf_t* buf);
size_t buf_write(buf_t* buf, void* data, size_t data_size);
void buf_empty(buf_t* buf);
void buf_free(buf_t* buf);
