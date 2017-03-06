#include "Poco/Exception.h"
#include "Poco/Stopwatch.h"
#include "Poco/StreamCopier.h"

#include "Poco/Net/HTTPSStreamFactory.h"
#include "Poco/Net/HTTPStreamFactory.h"
#include "Poco/URI.h"
#include "Poco/URIStreamOpener.h"

#include "Poco/Net/AcceptCertificateHandler.h"
#include "Poco/Net/Context.h"
#include "Poco/Net/InvalidCertificateHandler.h"
#include "Poco/Net/SSLManager.h"

#include "Poco/JSON/Parser.h"
#include "Poco/NumberFormatter.h"

#include "Poco/Format.h"
#include "Poco/Mutex.h"
#include "Poco/Runnable.h"
#include "Poco/ThreadPool.h"

#include <iostream>
#include <memory>
#include <queue>

using Poco::JSON::Object;
using Poco::JSON::Array;
using Poco::JSON::Parser;
using Poco::DynamicStruct;
using Poco::Dynamic::Var;
using Poco::Runnable;
using Poco::NumberFormatter;
using Poco::ThreadPool;
using Poco::Mutex;
using Poco::format;
using Poco::Stopwatch;

using namespace Poco::Net;
using namespace std;

queue<int> uids_q;
Mutex      uids_m;
Mutex      out_m;

// fetcher
class VkFetcher : public Runnable {

    // thread loop
    virtual void run() {

        while (true) {
            string uids       = "";
            int    max_length = 2000;
            {
                uids_m.lock();
                // pop some uids from queue up to particular string length
                while (!uids_q.empty() && uids.length() < max_length) {
                    uids += to_string(uids_q.front()) + ",";
                    uids_q.pop();
                }
                uids_m.unlock();
            }
            // if uids not empty - call api
            if (uids.length())
                callApi(uids);

            // exit if queue empty
            if (uids_q.empty())
                break;
        }
        // unlock
    }

    // call vk api
    void callApi(const string& uids) {

        Parser parser;
        try {
            string domain = "https://api.vk.com/method/users.get";
            string fields = "country,sex,bdate";
            string url =
                format("%s?v=3&user_ids=%s&fields=%s", domain, uids, fields);
            auto& opener = Poco::URIStreamOpener::defaultOpener();
            auto  uri    = Poco::URI{url};
            auto  is     = std::shared_ptr<std::istream>{opener.open(uri)};

            Var           result = parser.parse(*is);
            Object::Ptr   obj    = result.extract<Object::Ptr>();
            DynamicStruct ds     = *obj;

            for (auto&& u : ds["response"]) {
                string uid, bdate, sex, cntr, result;
                uid    = u["uid"].isEmpty() ? "-" : u["uid"].toString();
                bdate  = u["bdate"].isEmpty() ? "-" : u["bdate"].toString();
                sex    = u["sex"].isEmpty() ? "-" : u["sex"].toString();
                cntr   = u["country"].isEmpty() ? "-" : u["country"].toString();
                result = format("%s\t%s\t%s\t%s", uid, sex, bdate, cntr);
                out_m.lock();
                cout << result << endl;
                out_m.unlock();
            }
            // Poco::StreamCopier::copyToString(*(is.get()), content);
        } catch (Poco::Exception& e) {
            std::cerr << e.displayText() << std::endl;
        }
    }

    void parse() {}
};

// initialize SSL
void initialize() {

    HTTPStreamFactory::registerFactory();
    HTTPSStreamFactory::registerFactory();

    // http://stackoverflow.com/questions/18315472/https-request-in-c-using-poco
    initializeSSL();
    SSLManager::InvalidCertificateHandlerPtr ptrHandler(
        new AcceptCertificateHandler(false));
    Context::Ptr ptrContext(new Context(Context::CLIENT_USE, ""));
    SSLManager::instance().initializeClient(0, ptrHandler, ptrContext);
}

int main(int argc, char** argv) {

    initialize();

    // load ids
    for (uint32_t i = 0; i < 100000; i++) {
        uids_q.push(i);
    }

    // launch threads
    auto&& pool = Poco::ThreadPool::defaultPool();
    // default capacity 16
    pool.addCapacity(24 - pool.capacity());
    Poco::Stopwatch sw;
    sw.start();
    VkFetcher fetcher;
    for (int i = 0; i < 24; i++) {
        pool.start(fetcher);
    }
    pool.joinAll();
    sw.stop();
    cerr << "Elapsed: " << sw.elapsedSeconds() << "s" << endl;
    return 0;
}
