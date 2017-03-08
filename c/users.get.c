
// ॐ //

#include <locale.h>
#include <pthread.h>
#include <stdio.h>

#include "worker.h"

#define THREAD_NUM 1

int main(int argc, char** argv) {

    // setlocale(LC_ALL, "ru_RU.UTF-8");
    setlocale(LC_ALL, "");

    pthread_t threads[THREAD_NUM];
    for (long tid = 0; tid < THREAD_NUM; tid++) {
        int rc = pthread_create(&threads[tid], NULL, worker, (void*)tid);
        if (rc) {
            printf("ERROR %d", rc);
            return -1;
        }
    }

    for (long tid = 0; tid < THREAD_NUM; tid++) {
        pthread_join(threads[tid], NULL);
        fprintf(stderr, "Thread %d end\n", tid);
    }
    pthread_exit(NULL);
}
