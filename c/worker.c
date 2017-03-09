
// ॐ //

#include <curl/curl.h>
#include <curl/multi.h>
#include <errno.h>
#include <math.h>
#include <string.h>
#include <unistd.h>

#include "buf.h"
#include "json.h"
#include "worker.h"

#define API                                                                    \
    "https://api.vk.com/method/"                                               \
    "users.get?v=3&fields=bdate,city,country,sex&uids="
#define API_LEN sizeof API

const int  max_cnt         = 800;
const int  max_len         = 4000;
const long uids_per_thread = 1000000;
const long max_con         = 10;

// UTILITY //

// Generate VK url with sequental uids
static unsigned long generate_url(char* url, unsigned long from) {
    sprintf(url, "%s", API);
    int len  = API_LEN - 1;
    int cnt  = 0;
    int nlen = 0;
    while (cnt++ < max_cnt && len < max_len) {
        nlen = (int)floor(log10(from)) + 2;
        sprintf(url + len, "%d,", from++);
        len += nlen;
    }
    return from;
}

// Callback for curl, writes data to buffer
static size_t on_chunk(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t realsize = size * nmemb;
    buf_t* buf      = (buf_t*)userp;
    buf_write(buf, contents, realsize);
    return realsize;
}

/////////////////////////////////////////
// ONE KEEP-ALIVE DOWNLODAD PER THREAD //
/////////////////////////////////////////

void* worker(void* tid) {

    CURL*    curl;
    CURLcode res;

    curl = curl_easy_init();
    if (curl) {
        /*
        curl_easy_setopt(curl, CURLOPT_STDERR, stdout);
        curl_easy_setopt(curl, CURLOPT_VERBOSE, 1L);
        curl_easy_setopt(curl, CURLOPT_HEADER, 1L);
        */

        buf_t* buf = buf_new();
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, on_chunk);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, buf);

        char url[max_len + 100];

        unsigned long from = ((long)tid) * uids_per_thread + 1;
        unsigned long to   = from + uids_per_thread + 1;

        while (from < to) {
            from = generate_url(url, from);
            curl_easy_setopt(curl, CURLOPT_URL, url);
            res = curl_easy_perform(curl);

            // process buffer
            parse_json(buf);
            // cleanup
            buf_empty(buf);
            if (res != CURLE_OK)
                fprintf(stderr, curl_easy_strerror(res));
        }
        buf_free(buf);
        curl_easy_cleanup(curl);
    }
    return NULL;
}

///////////////////////////////////
// MULTIPLE DOWNLODAD PER THREAD //
///////////////////////////////////

void* worker_multi(void* tid) {

    CURLM* curlm;
    CURL*  curl;

    curlm = curl_multi_init();
    curl_multi_setopt(curlm, CURLMOPT_MAXCONNECTS, max_con);

    buf_t* bufs[max_con];
    char*  urls[max_con];

    unsigned long from = ((long)tid) * uids_per_thread + 1;
    unsigned long to   = from + uids_per_thread + 1;

    for (int i = 0; i < max_con; i++) {
        curl = curl_easy_init();
        if (curl != NULL) {

            bufs[i] = buf_new();
            urls[i] = malloc(max_len + 100);
            from    = generate_url(urls[i], from);

            curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, on_chunk);
            curl_easy_setopt(curl, CURLOPT_WRITEDATA, bufs[i]);
            curl_easy_setopt(curl, CURLOPT_PRIVATE, i);
            curl_easy_setopt(curl, CURLOPT_HEADER, 0L);
            curl_easy_setopt(curl, CURLOPT_VERBOSE, 0L);
            curl_easy_setopt(curl, CURLOPT_URL, urls[i]);
        } else {
            fprintf(stderr, "ERRO curl easy\n");
        }
        curl_multi_add_handle(curlm, curl);
    }

    fd_set         R, W, E;
    int M;
    long L;
    struct timeval T;
    // wait finish
    int curls_cnt = -1;
    while (curls_cnt) {

        curl_multi_perform(curlm, &curls_cnt);
        // fprintf(stderr, "tid: %d, curls: %d\n", tid, curls_cnt);
        if (curls_cnt) {
            FD_ZERO(&R);
            FD_ZERO(&W);
            FD_ZERO(&E);

            if (curl_multi_fdset(curlm, &R, &W, &E, &M)) {
                fprintf(stderr, "E: curl_multi_fdset\n");
                continue;
            }

            if (curl_multi_timeout(curlm, &L)) {
                fprintf(stderr, "E: curl_multi_timeout\n");
                continue;
            }
            if (L == -1)
                L = 100;

            if (M == -1) {
                sleep((unsigned int)L / 1000);
            } else {
                T.tv_sec  = L / 1000;
                T.tv_usec = (L % 1000) * 1000;

                if (0 > select(M + 1, &R, &W, &E, &T)) {
                    fprintf(stderr, "E: select(%i,,,,%li): %i: %s\n", M + 1, L,
                            errno, strerror(errno));
                    continue;
                }
            }
        }

        // iterate over messages
        CURLMsg* msg;
        int      msg_cnt;
        while ((msg = curl_multi_info_read(curlm, &msg_cnt))) {

            if (msg->msg == CURLMSG_DONE) {

                CURL* curl = msg->easy_handle;
                long  i    = -1;

                curl_easy_getinfo(curl, CURLINFO_PRIVATE, &i);

                // fprintf(stderr, "%d-%d done\n", (long)tid, i);

                if (i >= 0 && i < max_con) {
                    if (bufs[i]->size > 0) {
                        parse_json(bufs[i]);
                        buf_empty(bufs[i]);
                    } else {
                        fprintf(stderr, "thread %d, bufs[%d] is empty\n", tid,
                                i);
                    }
                }

                if (from < to) {
                    // add new url
                    from = generate_url(urls[i], from);

                    curl_multi_remove_handle(curlm, curl);
                    curl_easy_setopt(curl, CURLOPT_URL, urls[i]);
                    curl_multi_add_handle(curlm, curl);
                    continue;
                }
                // cleanup
                curl_multi_remove_handle(curlm, curl);
                curl_easy_cleanup(curl);
            } else {
                fprintf(stderr, "E: CURL msg %d", msg->msg);
            }
        }
    }

    // cleanup
    for (int i = 0; i < max_con; i++) {
        buf_free(bufs[i]);
        free(urls[i]);
    }
    curl_multi_cleanup(curlm);
}
