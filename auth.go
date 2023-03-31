package googleplay

import (
   "github.com/googleplay-persistent/rosso/http"
   "github.com/googleplay-persistent/rosso/protobuf"
   "github.com/googleplay-persistent/rosso/tls"
   "bufio"
   "io"
   "net/url"
   "os"
   "strconv"
   "strings"
)

func (a *Auth) Exchange() error {
   // these values take from Android API 28
   body := url.Values{
      "Token": {a.Get_Token()},
      "app": {"com.android.vending"},
      "client_sig": {"38918a453d07199354f8b19af05ec6562ced5788"},
      "service": {"oauth2:https://www.googleapis.com/auth/googleplay"},
   }.Encode()
   req := http.Post()
   req.Body_String(body)
   req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   req.URL.Host = "android.googleapis.com"
   req.URL.Path = "/auth"
   req.URL.Scheme = "https"
   res, err := http.Default_Client.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   a.Values = read_query(res.Body)
   return nil
}
func (h Header) Set_Agent(head http.Header) {
   var b []byte
   // `sdk` is needed for `/fdfe/delivery`
   b = append(b, "Android-Finsky (sdk="...)
   // valid range 0 - 0x7FFF_FFFF
   b = strconv.AppendInt(b, 9, 10)
   // com.android.vending
   b = append(b, ",versionCode="...)
   if h.Single {
      // valid range 8032_0000 - 8091_9999
      b = strconv.AppendInt(b, 8091_9999, 10)
   } else {
      // valid range 8092_0000 - 0x7FFF_FFFF
      b = strconv.AppendInt(b, 9999_9999, 10)
   }
   b = append(b, ')')
   head.Set("User-Agent", string(b))
}

func (h Header) Set_Device(head http.Header) error {
   id, err := h.Device.ID()
   if err != nil {
      return err
   }
   head.Set("X-DFE-Device-ID", strconv.FormatUint(id, 16))
   return nil
}

func (h Header) Set_Auth(head http.Header) {
   head.Set("Authorization", "Bearer " + h.Auth.Get_Auth())
}

func read_query(read io.Reader) url.Values {
   values := make(url.Values)
   scan := bufio.NewScanner(read)
   for scan.Scan() {
      key, value, pass := strings.Cut(scan.Text(), "=")
      if pass {
         values.Add(key, value)
      }
   }
   return values
}

type Header struct {
   Auth Auth // Authorization
   Device Device // X-DFE-Device-ID
   Single bool
}

func (h *Header) Open_Auth(name string) error {
   file, err := os.Open(name)
   if err != nil {
      return err
   }
   defer file.Close()
   h.Auth.Values = read_query(file)
   return nil
}

func (h *Header) Open_Device(name string) error {
   data, err := os.ReadFile(name)
   if err != nil {
      return err
   }
   h.Device.Message, err = protobuf.Unmarshal(data)
   if err != nil {
      return err
   }
   return nil
}

type Auth struct {
   url.Values
}

type Response struct {
   *http.Response
}

func (r Response) Create(name string) error {
   file, err := os.Create(name)
   if err != nil {
      return err
   }
   defer file.Close()
   if _, err := file.ReadFrom(r.Body); err != nil {
      return err
   }
   return nil
}

func (a Auth) Get_Auth() string {
   return a.Get("Auth")
}

func (a Auth) Get_Token() string {
   return a.Get("Token")
}

// You can also use host "android.clients.google.com", but it also uses
// TLS fingerprinting.
func New_Auth(email, passwd string) (*Response, error) {
   // Client_Hello
   hello, err := tls.Parse(tls.Android_API)
   if err != nil {
      return nil, err
   }
   client := http.Default_Client
   client.Transport = hello.Transport()
   body := url.Values{
      "Email": {email},
      "Passwd": {passwd},
      "client_sig": {""},
      // wikipedia.org/wiki/URL_encoding#Types_of_URI_characters
      "droidguard_results": {"-"},
   }.Encode()
   req := http.Post()
   req.Body_String(body)
   req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   req.URL.Host = "android.googleapis.com"
   req.URL.Path = "/auth"
   req.URL.Scheme = "https"
   res, err := client.Do(req)
   if err != nil {
      return nil, err
   }
   return &Response{res}, nil
}

