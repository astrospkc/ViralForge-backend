package handlers

import (
	"fmt"
	"strconv"
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
)

// create 1st comment
// reply comment
// delete comment
// edit comment
// like comments
// unlike comment

type CreateCommentResponse struct{
	Data 	models.Comment  
	Success bool 
	Code    int
	Message string 
}

func CreateComment() fiber.Handler{
	return func(c fiber.Ctx) error {
		user_id ,_ := FetchUserId(c)

		var req_body models.Comment 
		if err := c.Bind().Body(&req_body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(CreateCommentResponse{
				Success: false,
				Code:  400,
				Message: fmt.Sprintf("Error while fetching body : ", err),
			})
		}

		v_id,_:= strconv.Atoi(c.Params("v_id"))
		video_id := int64(v_id)

		comment := &models.Comment{
			VideoID : video_id,
			UserID: user_id,
			ParentCommentID: req_body.ParentCommentID,
			RootCommentID: req_body.RootCommentID,
			Content : req_body.Content,
			LikeCount: 0,
		}

		err := connect.Db.NewInsert().Model(comment).Returning("*").Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(CreateCommentResponse{
				Success: false,
				Code:  400,
				Message: fmt.Sprintf("Error while creating comment : ", err),
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(CreateCommentResponse{
			Data: *comment,
			Success: false,
			Code:  400,
			Message: fmt.Sprintf("Error while creating comment : ", err),
		})
		
	}
}

type UpdateCommentResponse struct {
	Success bool 
	Code    int 
	Message string
}

func UpdateComment() fiber.Handler{
	return func (c fiber.Ctx) error{
		comment_id, _ := strconv.Atoi(c.Params("comment_id"))
		c_id := int64(comment_id)



		var body  struct{
			Content  string `json:"string"`

		}

		if err:= c.Bind().Body(&body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(UpdateCommentResponse{
				Success: false,
				Code :400,
				Message: fmt.Sprintf("Error while fetching data from request body : ", err),
			})
		}

		err := connect.Db.NewUpdate().Model((*models.Comment)(nil)).Set("content = ?", body.Content).Where("id = ?", c_id).Returning("*").Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(UpdateCommentResponse{
				Code:400,
				Message: fmt.Sprintf("Error while editing comment: ", err),
				Success: false,
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(UpdateCommentResponse{
			Code:200,
			Message:"Successfully edited comment",
			Success: true,
		})
		
	}
}

func DeleteComment() fiber.Handler{
	return func (c fiber.Ctx) error{
		comment_id,_:=strconv.Atoi(c.Params("comment_id"))

		_,err := connect.Db.NewUpdate().Model((*models.Comment)(nil)).Set("status = ?", models.CommentHidden).Where("id = ?",comment_id).Exec(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"Message":fmt.Sprintf("Error while deleting comment: ", err),
				"Code":400,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"Message":fmt.Sprintf(" Successfully deleted comment"),
			"Code":200,
		})
		
	}
}

type GetCommentsResponse struct{
	Data	[]models.Comment 
	Success bool 
	Code    int 
	Messaage string
}

// BUT WHEN SENDING COMMENTS ARRAY , WE NEED TO SEND THE COMMENTS AS WELL AS USER INFO: JOIN THE USER AND COMMENTS TABLE TO FETCH THE INFO.


func GetComments() fiber.Handler{
	return func (c fiber.Ctx) error{
		video_id, _ := strconv.Atoi(c.Params("v_id"))
		v_id:= int64(video_id)

		var comments []models.Comment 

		err := connect.Db.NewSelect().Model(&comments).Relation("User").Where("video_id = ?", v_id).Where("status = ?", models.CommentVisible).Order("created_at ASC").Scan(c.Context())
		if err!=nil{
			fmt.Printf("error: ", err)
			return c.Status(fiber.StatusBadRequest).JSON(GetCommentsResponse{
				Success:false,
				Code: 400,
				Messaage: "Failed to get comments",
			})
		}

		return c.Status(fiber.StatusOK).JSON(GetCommentsResponse{
			Data:    comments,
			Success: true,
			Code:200,
			Messaage: "Successfully fetched all the comments",
		})
		
	}
}



func GetReplies() fiber.Handler{
	return func(c fiber.Ctx) error {
		// get parent comment id
		comment_id, err := strconv.Atoi(c.Params("comment_id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(GetCommentsResponse{
				Success:  false,
				Code:     400,
				Messaage: "Invalid comment ID",
			})
		}
		c_id := int64(comment_id)

		var replies []models.Comment
		// Also fetch the user associated with each reply
		err = connect.Db.NewSelect().
			Model(&replies).
			Relation("User").
			Where("parent_comment_id = ?", c_id).
			Where("status = ?", models.CommentVisible). // Only get visible comments
			Order("created_at ASC").                   // Show oldest replies first
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(GetCommentsResponse{
				Success:  false,
				Code:     500,
				Messaage: fmt.Sprintf("Failed to fetch replies: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(GetCommentsResponse{
			Data:     replies,
			Success:  true,
			Code:     200,
			Messaage: "Successfully fetched all replies",
		})
	}
}

// type Reviews struct{
// 	id  int64;
//     userId int64;
//     userName string;
//     avatar  string;
//     rating  int16;
//     comment []string;
// }

// func GetReviews() fiber.Handler{
// 	return func(c fiber.Ctx) error{
// 		var reviews []Reviews 
// 		video_id, _ := strconv.Atoi(c.Params("v_id"))
// 		v_id:= int64(video_id)
		



// 		var comments []models.Comment
// 		err := connect.Db.NewSelect().Model(&comments).Where("video_id = ?", v_id).Scan(c.Context())
// 		if err!=nil{
// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 				"message":"failed to fetch reviews",
// 				"code":400,
// 			})
// 		}
// 		// user info:
// 		var user_info models.User
// 		err = connect.Db.NewSelect().Model(&user_info).Where("id = ?", user_id).Scan(c.Context())
// 		if err!=nil{
// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 				"message":"failed to fetch reviews",
// 				"code":400,
// 			})
// 		}

// 		reviews = append(reviews,)
		


		

// 	}
// }