#include "buf.h"

#include <string.h>
#include <stdio.h>
 
// Allocates buf on heap and init it
buf_t* Buf_new() {
    buf_t* buf = malloc(sizeof(buf_t));
    Buf_init(buf);
    return buf;
}

// Allocates memory in buf
void Buf_init(buf_t* buf) {
    buf->buf = malloc(1);
    buf->size = 0;
    buf->buf[0] = 0;
}

// Write new chunk of Data To buf
size_t Buf_write(buf_t* buf, void* data, size_t data_size) {
    // +1 byte for zero terminate
    buf->buf = realloc(buf->buf, buf->size + data_size + 1);
    if (buf->buf == NULL) {
        fprintf(stderr, "ERR: not enough memory to realocate buf");
        return 0;
    }
    // copy new data to the end of buff
    memcpy(&(buf->buf[buf->size]), data, data_size);
    // update size (exclude zero byte)
    buf->size += data_size;
    // zero terminate text
    buf->buf[buf->size] = 0;
    return buf->size;
}

// frees alocated memory and reinit
void Buf_empty(buf_t* buf) {
    free(buf->buf);
    Buf_init(buf);
}

// frees alcated memory and buffer itself
void Buf_free(buf_t* buf) {
    free(buf->buf);
    free(buf);
}
