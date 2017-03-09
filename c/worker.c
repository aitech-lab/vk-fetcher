
// ॐ //

#include <curl/curl.h>
#include <curl/multi.h>

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
const long uids_per_thread = 100000;
const long max_con         = 100;

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

static size_t on_chunk(void* contents, size_t size, size_t nmemb, void* userp) {
    size_t realsize = size * nmemb;
    buf_t* buf      = (buf_t*)userp;
    buf_write(buf, contents, realsize);
    return realsize;
}

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

/////////////////////////////////////////////////////////////

static void multi_init(CURLM* curlm, char* url, buf_t* buf) {
    CURL* curl = curl_easy_init();

    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, on_chunk);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, buf);

    curl_easy_setopt(curl, CURLOPT_HEADER, 0L);
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_PRIVATE, url);
    curl_easy_setopt(curl, CURLOPT_VERBOSE, 0L);

    curl_multi_add_handle(curlm, curl);
}

void* worker_multi(void* tid) {

    CURLM* curlm;

    curlm = curl_multi_init();
    curl_multi_setopt(curlm, CURLMOPT_MAXCONNECTS, max_con);
    buf_t*         bufs[max_con];
    char*         urls[max_con];
    unsigned long from = ((long)tid) * uids_per_thread + 1;
    for (int i = 0; i < max_con; i++) {
        bufs[i] = buf_new();
        urls[i] = malloc(max_len + 100);
        from    = generate_url(urls[i], from);
        multi_init(curlm, urls[i], bufs[i]);
    }

    // wait finish
    int curls_cnt = -1;
    while (curls_cnt) {

        curl_multi_perform(curlm, &curls_cnt);

        if (curls_cnt) {
        }

        CURLMsg* msg;
        int      msg_cnt;
        while ((msg = curl_multi_info_read(curlm, &msg_cnt))) {
            if (msg && (msg->msg == CURLMSG_DONE)) {
                CURL* curl = msg->easy_handle;
                curl_multi_remove_handle(curlm, curl);
                curl_easy_cleanup(curl);
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
