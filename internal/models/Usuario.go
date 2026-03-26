package models

type Usuario struct {

  Id int `json:"Id"`

  Nombre string `json:"Nombre"`

  Email string `json:"Email"`

  Perfil Profile `json:"Perfil"`

}
