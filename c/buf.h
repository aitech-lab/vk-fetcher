#pragma once

// An dynamic null terminated buffer

#include <stdlib.h>

typedef struct {
    char*  buf;
    size_t size;
} buf_t;

buf_t* Buf_new();
void Buf_init(buf_t* buf);
size_t Buf_write(buf_t* buf, void* data, size_t data_size);
void Buf_empty(buf_t* buf);
void Buf_free(buf_t* buf);
