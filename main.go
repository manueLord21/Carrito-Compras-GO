package main

//libresias utilizadas
import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

/*estructuras para la conexion de la base de datos*/

type Productos struct {
	Id       int
	Nombre   string
	Costo    int
	Cantidad int
	Archivo  string
}

type Lista struct {
	Producto  int
	Nombre    string
	Cantidad  int
	Costo     int
	Resultado int
	Archivo   string
}

type Pedidos struct {
	Id    int
	Costo int
	Fecha string
}

//Conexion a la base de datos desde PhP MyAdmin
func dbConn() (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := "root"
	dbPass := ""
	dbName := "carrito_db"
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	chackErr(err)
	return db
}

//La carpeta donde estan las paginas web
var temp = template.Must(template.ParseGlob("pagina/*"))

//Pagina principal
func index(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	query, err := db.Query("SELECT * FROM productos")
	chackErr(err)
	pro := Productos{}
	res := []Productos{}
	for query.Next() {
		var id, cantidad, costo int
		var nombre, archivo string
		err = query.Scan(&id, &nombre, &costo, &cantidad, &archivo)
		pro.Id = id
		pro.Nombre = nombre
		pro.Costo = costo
		pro.Archivo = archivo
		res = append(res, pro)
		chackErr(err)
	}
	temp.ExecuteTemplate(w, "index", res)
	defer db.Close()
}

//Función para insertar datos al carrito
func insertar(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	if r.Method == "POST" {
		r.ParseForm()
		o := r.Form["opt"]
		c := r.Form["cant"]
		n := r.Form["name"]
		m := r.Form["cost"]
		a := r.Form["arch"]
		for i := 0; i < len(o); i++ {
			insForm, err := db.Prepare("INSERT INTO lista(producto, nombre, archivo, cantidad, costo) VALUES(?,?,?,?,?)")
			chackErr(err)
			insForm.Exec(o[i], n[i], a[i], c[i], m[i])
			log.Println("INSERT: dato: " + n[i] + " [aceptado]")
		}
		defer db.Close()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

//Pagina de la lista de los productos que es agegaron al carrito
func lista(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	query, err := db.Query("SELECT producto, nombre, archivo, costo, SUM(cantidad) AS cantidad, SUM(cantidad*costo) AS resultado  FROM lista GROUP BY producto")
	chackErr(err)
	lista := Lista{}
	res := []Lista{}
	for query.Next() {
		var producto, cantidad, costo, resultado int
		var nombre, archivo string
		err = query.Scan(&producto, &nombre, &archivo, &costo, &cantidad, &resultado)
		lista.Producto = producto
		lista.Nombre = nombre
		lista.Cantidad = cantidad
		lista.Costo = costo
		lista.Archivo = archivo
		lista.Resultado = resultado
		res = append(res, lista)
		chackErr(err)
	}
	var total string
	db.QueryRow("SELECT SUM(cantidad*costo) FROM lista").Scan(&total)
	temp.ExecuteTemplate(w, "lista", res)
	fmt.Fprintln(w, "<p class='tot'>TOTAL: $ "+total+" MX</p>")
	defer db.Close()
}

//Resta la cantidad de los articulos, si llega a cero se elimina el articulo de la lista
func restar(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	res := r.URL.Query().Get("id")
	can := r.URL.Query().Get("can")
	d, err := strconv.Atoi(can)
	chackErr(err)
	d = d - 1
	if d == 0 {
		db.QueryRow("DELETE FROM lista WHERE producto=?", res)
	} else {
		up, err := db.Prepare("UPDATE lista SET cantidad=? WHERE producto=?")
		chackErr(err)
		up.Exec(d, res)
		fmt.Println(res, " - ", d)
	}
	defer db.Close()
	http.Redirect(w, r, "/lista", http.StatusSeeOther)
}

//Suma la cantidad de los articulos de la lista
func sumar(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	res := r.URL.Query().Get("id")
	can := r.URL.Query().Get("can")
	d, err := strconv.Atoi(can)
	chackErr(err)
	d = d + 1
	up, err := db.Prepare("UPDATE lista SET cantidad=? WHERE producto=?")
	chackErr(err)
	up.Exec(d, res)
	fmt.Println(res, " - ", d)
	defer db.Close()
	http.Redirect(w, r, "/lista", http.StatusSeeOther)
}

//Agerda los datos de la lista a otra taba llamada pedidas
func agregar(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	if r.Method == "POST" {
		db.QueryRow("INSERT INTO pedidos (costo) SELECT SUM(cantidad*costo) FROM lista")
		db.QueryRow("DELETE FROM lista")
		log.Println("[aceptado]")
		defer db.Close()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

//Elimama todos los datos de la lista de carrito
func eliminar(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	emp := r.URL.Query().Get("id")
	del, err := db.Prepare("DELETE FROM lista WHERE producto=?")
	chackErr(err)
	del.Exec(emp)
	log.Println("DELETE")
	defer db.Close()
	http.Redirect(w, r, "/lista", http.StatusSeeOther)
}

//Muestra la lista da los pedidos
func pedidos(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	query, err := db.Query("SELECT * FROM pedidos")
	chackErr(err)
	pe := Pedidos{}
	res := []Pedidos{}
	for query.Next() {
		var idpredido, costo int
		var fecha string
		err = query.Scan(&idpredido, &costo, &fecha)
		pe.Id = idpredido
		pe.Costo = costo
		pe.Fecha = fecha
		res = append(res, pe)
		chackErr(err)
	}
	temp.ExecuteTemplate(w, "pedidos", res)
	defer db.Close()
}

//Función pancipal de prgrama
func main() {
	//Impresión en terminal la pagina que es va a utilizar
	log.Println("Conectando en: http://localhost:8080")

	//Funcion para poder colocar archivos multimeda solo se pone la carpeta en donde van a estar colocados
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	//Paginas y Funciónes que es van a utilizar
	http.HandleFunc("/", index)
	http.HandleFunc("/insertar", insertar)
	http.HandleFunc("/agregar", agregar)
	http.HandleFunc("/pedidos", pedidos)
	http.HandleFunc("/restar", restar)
	http.HandleFunc("/sumar", sumar)
	http.HandleFunc("/eliminar", eliminar)
	http.HandleFunc("/lista", lista)

	//Conexión a la pagina señalado al extención
	http.ListenAndServe(":8080", nil)
}

//Función error para mostrar los errores
func chackErr(e error) {
	if e != nil {
		fmt.Println("Error: ", e)
	}
}
