package java;  

import org.apache.thrift.TException;  
import org.apache.thrift.protocol.TBinaryProtocol;  
import org.apache.thrift.protocol.TProtocol;  
import org.apache.thrift.transport.TSocket;  
import org.apache.thrift.transport.TTransport;  
import org.apache.thrift.transport.TTransportException;  

public class FaeClient {  

    public static void main(String[] args) {  

        try {  
            TTransport transport;  

            transport = new TSocket("localhost", 9001); 
            transport.open();  

            TProtocol protocol = new TBinaryProtocol(transport);  
            fun.rpc.Client client = new fun.rpc.Client(protocol);  

            System.out.println(client.noop(1));  

            transport.close();  
        } catch (TTransportException e) {  
            e.printStackTrace();  
        } catch (TException x) {  
            x.printStackTrace();  
        }  
    }  

}
