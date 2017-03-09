
// ॐ //

#pragma once

// one keepalive download per thread
void* worker(void* tid);

// mutliple downloads per thread
void* worker_multi(void* tid);
