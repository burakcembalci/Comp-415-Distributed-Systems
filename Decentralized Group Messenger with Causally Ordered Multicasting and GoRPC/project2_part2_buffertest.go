package main

import (
    "fmt"
    "net"
    "net/rpc"
    "os"
    "log"
    "bufio"
    "sync"
    "time"

)

const numPeers = 3
var argsWithoutProg = os.Args[1:]
var connectedPeers =0
var peerPort = argsWithoutProg[0]
var mutex = &sync.Mutex{}

var p peer



type peer struct {
    username string
    ip  string
    port string
    neighoursList []string 
    clientlist [] *rpc.Client
    peertimestamp [numPeers]int
    peerIndex int
    messageBuffer []*Message
}

type Message struct{
    Transcript string
    Oid string
    Timestamp [numPeers]int
    SenderIndex int
}

func checkError(err error) {
    if err != nil {
        fmt.Println("Fatal error ", err.Error())
        os.Exit(1)
    }
}

func generateUserName() string{
    ip := GetLocalIP()
    port := peerPort
    name := ip+":"+port
    return name 
}

func generateNeighboursList()  ([] string,int){
    var list []string
    myName := generateUserName()
    file, err := os.Open("peers.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    i:=0
    index := 0
    for scanner.Scan() {
        if (scanner.Text()!=myName){
            list = append(list,scanner.Text())
        }else{
            index = i
        }
    i= i+1
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
    return list, index
}


func GetLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return ""
    }
    for _, address := range addrs {
        // check the address type and if it is not a loopback the display it
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return ""
}
type M int 

func getMax(a int,b int) int {
    if(b>a){
        return b
    }
    return a
}



func client(){
    reader := bufio.NewReader(os.Stdin) // read in input from stdin
    input := "0"
    for input != "exit"{
        fmt.Printf(">Enter message:  \n")
        input, _ = reader.ReadString('\n')
        mutex.Lock()
        p.peertimestamp[p.peerIndex]= p.peertimestamp[p.peerIndex] +1 
        messagetimestamp := p.peertimestamp
       


        msg :=Message{Transcript:input,Oid:p.username,Timestamp:messagetimestamp,SenderIndex:p.peerIndex}
        mutex.Unlock()//ask where this should be this is a design choice

        for _,element := range p.clientlist{
             go sendMessage(msg,element)
        }
    }
}

func isOrdered(msg *Message) bool {
    //fmt.Println("fi msg time stamp",msg.Timestamp,msg.SenderIndex)
    //fmt.Println("fi p.peertimestamp",p.peertimestamp)
    //fmt.Println("first if ",msg.Timestamp[msg.SenderIndex],(p.peertimestamp[msg.SenderIndex]+1))
    if(msg.Timestamp[msg.SenderIndex]==(p.peertimestamp[msg.SenderIndex]+1)){
        for index,_ := range p.peertimestamp{
            if(index != msg.SenderIndex){
            //    fmt.Println("second if",p.peertimestamp[index],msg.Timestamp[index])
                if(p.peertimestamp[index]< msg.Timestamp[index]){
                    return false 
                }
            }
        }
    }else{
        return false
    }
    return true
}       
    

func sendMessage(msg Message, client *rpc.Client){
    var reply int 
    err :=client.Call("M.MessagePost",msg,&reply)
    if err!=nil{
        log.Fatal("send error :",err)
    }
      fmt.Println("Message sucessfully sent",reply)


}
 
func updateTimestamp(msg *Message){
    for index,_ := range p.peertimestamp{
        p.peertimestamp[index]= getMax(p.peertimestamp[index],msg.Timestamp[index])

    }
    fmt.Println("",p.peertimestamp)
}

func (t *M) MessagePost(msg *Message,reply *int) error {
    if(msg.Transcript=="bekle\n"&& p.peerIndex==2){
        time.Sleep(30*time.Second)
    }
    //fmt.Println(" msg time stamp",msg.Timestamp,msg.SenderIndex)
    //fmt.Println(" p.peertimestamp",p.peertimestamp)
   // fmt.Println(msg)
   // fmt.Println(p)
    
    if isOrdered(msg){
    mutex.Lock()
    updateTimestamp(msg)
    //fmt.Println("Timestamp:",p.peertimestamp)
    fmt.Println("Message recieved from:"+msg.Oid+"Content:"+msg.Transcript,msg.Timestamp)
    //fmt.Println("Enter message:")
    *reply=1
    mutex.Unlock()
    }else{
        mutex.Lock()
       // fmt.Println("Adding message to buffer")
      //  fmt.Println(p.messageBuffer)
        p.messageBuffer=append(p.messageBuffer,msg)
       // fmt.Println("Added message to buffer")
      //  fmt.Println(p.messageBuffer)
        mutex.Unlock()
        for{
            time.Sleep(1 * time.Second)
            mutex.Lock()
            if(isOrdered(msg)){
               // fmt.Println(p.messageBuffer)
               // fmt.Println("Removing message to buffer")//what kind of a language has an arraylist that does not contain remove element operation !!!!
                i:=99
                for index,element := range p.messageBuffer{
                    if element == msg{
                        i=index
                    }
                }
                p.messageBuffer = append(p.messageBuffer[:i], p.messageBuffer[i+1:]...)
              //  fmt.Println(p.messageBuffer)
                updateTimestamp(msg)
                *reply=1
                fmt.Println("Message recieved from:"+msg.Oid+"Content:"+msg.Transcript)
               // fmt.Println("Enter message:") 
                mutex.Unlock()
                return nil
                }
            mutex.Unlock()




        }
    }
    return nil
}

func service(){
//recieve messages


    // creating and regestring Arith Service on the network
    magic := new(M)
    rpc.Register(magic)

    // sarting server
    tcpAddr, err := net.ResolveTCPAddr("tcp", generateUserName())
    checkError(err)
    listener, err := net.ListenTCP("tcp", tcpAddr)
    checkError(err)

    fmt.Println(">Serivice is running.... ")

    // similar to server code
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }else{
             go rpc.ServeConn(conn)
        }
    }

}


func main(){
    //connect to services
    var clist [] *rpc.Client
    nl,in := generateNeighboursList()
    var ts [numPeers]int
    var buffer []*Message
    /*
    p :=peer{
    username:generateUserName(),
    ip:GetLocalIP(),
    port:peerPort,
    neighoursList:nl,
    clientlist:clist,
    peertimestamp: ts,
    peerIndex:in,
    messageBuffer:buffer}
    */

    p.username=generateUserName()
    p.ip=GetLocalIP()
    p.port=peerPort
    p.neighoursList=nl
    p.clientlist=clist
    p.peertimestamp=ts
    p.peerIndex=in
    p.messageBuffer=buffer


    fmt.Println(p.peertimestamp)
    go service()
    reader := bufio.NewReader(os.Stdin)
    _, _ = reader.ReadString('\n')

    for _,element := range p.neighoursList{
            client, err := rpc.Dial("tcp", element) // connecting to the service
            if(err!=nil){
                 log.Fatal("Dial error :",err)
            }else{
            clist = append(clist,client)
        }
    }
    p.clientlist = clist
    


    go client()
    


    for{

        

    }

}