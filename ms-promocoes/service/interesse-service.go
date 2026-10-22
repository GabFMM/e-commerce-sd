package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"e-commerce-sd/data"
)

var categoriasValidas = map[string]bool{"A": true, "B": true, "C": true}

func normalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// normalizarCategorias filtra as inválidas e remove duplicadas.
func normalizarCategorias(categorias []string) []string {
	vistas := map[string]bool{}
	validas := []string{}
	for _, cat := range categorias {
		c := strings.ToUpper(strings.TrimSpace(cat))
		if categoriasValidas[c] && !vistas[c] {
			vistas[c] = true
			validas = append(validas, c)
		}
	}
	return validas
}

// CadastrarInteresse define a lista de categorias do e-mail (substitui a anterior).
func CadastrarInteresse(email string, categorias []string) error {
	emailNorm := normalizarEmail(email)
	if emailNorm == "" {
		log.Printf("[AVISO-PROMOCOES] Cadastro ignorado: e-mail vazio")
		return nil
	}

	validas := normalizarCategorias(categorias)

	_, err := data.Pool("PROMOCOES").Exec(context.Background(), `
		INSERT INTO interesses (email, categorias)
		VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET categorias = EXCLUDED.categorias
	`, emailNorm, validas)
	if err != nil {
		return fmt.Errorf("salvar interesse: %w", err)
	}

	log.Printf("[INFO-PROMOCOES] Interesse registrado. Email: %s | Categorias: %v", emailNorm, validas)
	return nil
}

// RemoverInteresse apaga o cadastro do e-mail.
func RemoverInteresse(email string) error {
	emailNorm := normalizarEmail(email)

	res, err := data.Pool("PROMOCOES").Exec(context.Background(),
		`DELETE FROM interesses WHERE email = $1`, emailNorm)
	if err != nil {
		return fmt.Errorf("remover interesse: %w", err)
	}

	if res.RowsAffected() == 0 {
		log.Printf("[INFO-PROMOCOES] Cancelamento ignorado: %s não estava cadastrado", emailNorm)
		return nil
	}
	log.Printf("[INFO-PROMOCOES] Interesse removido. Email: %s", emailNorm)
	return nil
}

// ObterEmailsPorCategoria devolve os e-mails que têm a categoria na lista.
func ObterEmailsPorCategoria(categoria string) ([]string, error) {
	cat := strings.ToUpper(strings.TrimSpace(categoria))

	rows, err := data.Pool("PROMOCOES").Query(context.Background(),
		`SELECT email FROM interesses WHERE $1 = ANY(categorias)`, cat)
	if err != nil {
		return nil, fmt.Errorf("consultar interessados: %w", err)
	}
	defer rows.Close()

	emails := []string{}
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}
