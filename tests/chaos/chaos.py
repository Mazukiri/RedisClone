import socket
import threading
import time
import random
import sys

HOST = 'localhost'
PORT = 8082

def fuzz_packet():
    """Sends garbage bytes to the server."""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(1)
        s.connect((HOST, PORT))
        garbage = bytearray(random.getrandbits(8) for _ in range(100))
        s.sendall(garbage)
        s.close()
    except Exception:
        pass

def flood_connections(count=500):
    """Opens many connections and holds them open."""
    conns = []
    print(f"Flooding with {count} connections...")
    for _ in range(count):
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.connect((HOST, PORT))
            conns.append(s)
        except Exception as e:
            print(f"Connection failed: {e}")
            break
    time.sleep(2)
    print("Closing connections...")
    for s in conns:
        try:
            s.close()
        except:
            pass

def spam_commands(count=1000):
    """Spams valid commands rapidly."""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.connect((HOST, PORT))
        for i in range(count):
            cmd = f"*3\r\n$3\r\nSET\r\n$5\r\nchaos\r\n$5\r\n{i}\r\n"
            s.sendall(cmd.encode())
            # We don't wait for response to stress the buffer
        s.close()
    except Exception as e:
        print(f"Spam failed: {e}")

def main():
    print("Starting Chaos Test...")
    
    threads = []
    
    # 1. Fuzzing
    print("Layer 1: Fuzzing")
    for _ in range(10):
        t = threading.Thread(target=fuzz_packet)
        t.start()
        threads.append(t)
        
    # 2. Connection Flooding
    print("Layer 2: Connection Flood")
    t = threading.Thread(target=flood_connections, args=(500,))
    t.start()
    threads.append(t)
    
    # 3. Command Spam
    print("Layer 3: command Spam")
    for _ in range(5):
        t = threading.Thread(target=spam_commands)
        t.start()
        threads.append(t)
        
    for t in threads:
        t.join()
        
    print("Chaos Test Finished. Check server logs for panic.")

if __name__ == "__main__":
    main()
