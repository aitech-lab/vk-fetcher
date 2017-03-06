add_executable(http.get http.get.cpp )
target_link_libraries(http.get PocoNetSSL PocoCrypto PocoNet PocoUtil PocoJSON PocoXML PocoFoundation )

add_executable(https.get https.get.cpp )
target_link_libraries(https.get PocoNetSSL PocoCrypto PocoNet PocoUtil PocoJSON PocoXML PocoFoundation )
