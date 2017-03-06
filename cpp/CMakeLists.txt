add_executable(users.get users.get.cpp )
target_link_libraries(users.get PocoNetSSL PocoCrypto PocoNet PocoUtil PocoJSON PocoXML PocoFoundation )
